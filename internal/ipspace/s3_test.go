package ipspace

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithy "github.com/aws/smithy-go"
	"github.com/stretchr/testify/require"
)

// fakeS3 is an in-memory S3 keyed by object key. By default it honors
// conditional writes (If-None-Match / If-Match); ignoreConditions=true models a
// provider that silently ignores them (older MinIO), and so should be rejected.
type fakeS3 struct {
	objs             map[string][]byte
	etags            map[string]string
	getErr           error
	putErr           error
	ignoreConditions bool // simulate a provider that ignores conditional headers
	rejectIfNoneMatch bool // simulate B2: errors when If-None-Match is present
	lastPut          *s3.PutObjectInput
}

func newFakeS3() *fakeS3 { return &fakeS3{objs: map[string][]byte{}, etags: map[string]string{}} }

func (f *fakeS3) GetObject(ctx context.Context, in *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	obj, ok := f.objs[*in.Key]
	if !ok {
		return nil, &smithy.GenericAPIError{Code: "NoSuchKey", Message: "missing"}
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(obj)), ETag: aws.String(f.etags[*in.Key])}, nil
}

func (f *fakeS3) PutObject(ctx context.Context, in *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	f.lastPut = in
	if f.putErr != nil {
		return nil, f.putErr
	}
	if in.IfNoneMatch != nil && f.rejectIfNoneMatch {
		return nil, &smithy.GenericAPIError{Code: "NotImplemented", Message: "If-None-Match not supported"}
	}
	key := *in.Key
	_, exists := f.objs[key]
	if !f.ignoreConditions {
		if in.IfNoneMatch != nil && exists {
			return nil, &smithy.GenericAPIError{Code: "PreconditionFailed", Message: "exists"}
		}
		if in.IfMatch != nil && f.etags[key] != *in.IfMatch {
			return nil, &smithy.GenericAPIError{Code: "PreconditionFailed", Message: "etag mismatch"}
		}
	}
	body, _ := io.ReadAll(in.Body)
	f.objs[key] = body
	f.etags[key] = `"` + versionOf(body)[:8] + `"`
	return &s3.PutObjectOutput{ETag: aws.String(f.etags[key])}, nil
}

func (f *fakeS3) DeleteObject(ctx context.Context, in *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	delete(f.objs, *in.Key)
	delete(f.etags, *in.Key)
	return &s3.DeleteObjectOutput{}, nil
}

func s3b(api s3API) S3Backend { return S3Backend{API: api, Bucket: "b", Key: "k", Label: "S3 b/k"} }

func TestS3GetMissingIsNil(t *testing.T) {
	blob, ver, err := s3b(newFakeS3()).Get(context.Background())
	require.NoError(t, err)
	require.Nil(t, blob)
	require.Empty(t, ver)
}

func TestS3PutGetRoundTrip(t *testing.T) {
	b := s3b(newFakeS3())
	require.NoError(t, b.Put(context.Background(), []byte("hello"), ""))
	blob, ver, err := b.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), blob)
	require.NotEmpty(t, ver)
}

func TestS3PutFirstUsesIfNoneMatch(t *testing.T) {
	f := newFakeS3()
	require.NoError(t, s3b(f).Put(context.Background(), []byte("x"), ""))
	require.NotNil(t, f.lastPut.IfNoneMatch)
	require.Equal(t, "*", *f.lastPut.IfNoneMatch)
	require.Nil(t, f.lastPut.IfMatch)
}

func TestS3PutUpdateUsesIfMatch(t *testing.T) {
	f := newFakeS3()
	f.objs["k"] = []byte("old")
	f.etags["k"] = `"etag123"`
	require.NoError(t, s3b(f).Put(context.Background(), []byte("x"), `"etag123"`))
	require.NotNil(t, f.lastPut.IfMatch)
	require.Equal(t, `"etag123"`, *f.lastPut.IfMatch)
	require.Nil(t, f.lastPut.IfNoneMatch)
}

// A real conditional-write CAS: a stale If-Match is refused as ErrRemoteChanged.
func TestS3PutStaleIfMatchIsRemoteChanged(t *testing.T) {
	f := newFakeS3()
	b := s3b(f)
	require.NoError(t, b.Put(context.Background(), []byte("v1"), ""))
	_, v1, _ := b.Get(context.Background())
	require.NoError(t, b.Put(context.Background(), []byte("v2"), v1)) // moves it on
	err := b.Put(context.Background(), []byte("v1-edit"), v1)         // stale
	require.ErrorIs(t, err, ErrRemoteChanged)
}

func TestS3PutConditionalConflictIsRemoteChanged(t *testing.T) {
	f := newFakeS3()
	f.putErr = &smithy.GenericAPIError{Code: "ConditionalRequestConflict", Message: "409"}
	err := s3b(f).Put(context.Background(), []byte("x"), "")
	require.ErrorIs(t, err, ErrRemoteChanged)
}

func TestS3GetOtherErrorPropagates(t *testing.T) {
	f := newFakeS3()
	f.getErr = &smithy.GenericAPIError{Code: "AccessDenied", Message: "nope"}
	_, _, err := s3b(f).Get(context.Background())
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrRemoteChanged))
}

// Pull/Push work over the S3 backend exactly as over local.
func TestS3PullPushViaBackend(t *testing.T) {
	b := s3b(newFakeS3())
	require.NoError(t, Push(context.Background(), b, "team", sampleEvents(), ""))
	got, ver, err := Pull(context.Background(), b, "team")
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.NotEmpty(t, ver)
}

// The conditional-write capability probe: it passes on a compliant provider and
// cleans up its probe object.
func TestS3VerifyConditionalWritesCompliant(t *testing.T) {
	f := newFakeS3()
	require.NoError(t, s3b(f).VerifyConditionalWrites(context.Background()))
	require.Empty(t, f.objs, "the probe object is cleaned up")
}

// A provider that silently ignores conditional headers is detected and rejected,
// rather than being trusted and silently losing writes.
func TestS3VerifyConditionalWritesIgnored(t *testing.T) {
	f := newFakeS3()
	f.ignoreConditions = true
	err := s3b(f).VerifyConditionalWrites(context.Background())
	require.Error(t, err)
	require.Empty(t, f.objs, "the probe object is cleaned up even on rejection")
}

// A provider that rejects the If-None-Match header outright (B2) is rejected.
func TestS3VerifyConditionalWritesHeaderRejected(t *testing.T) {
	f := newFakeS3()
	f.rejectIfNoneMatch = true
	err := s3b(f).VerifyConditionalWrites(context.Background())
	require.Error(t, err)
}

func TestNewS3BackendBuilds(t *testing.T) {
	b, err := NewS3Backend(context.Background(), S3Config{
		Endpoint: "https://example.r2.cloudflarestorage.com", Region: "auto",
		Bucket: "inv", Key: "meridian/ipspace.enc", AccessKey: "AK", Secret: "SK",
	})
	require.NoError(t, err)
	require.Contains(t, b.Describe(), "inv")
}
