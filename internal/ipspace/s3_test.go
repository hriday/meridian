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

// fakeS3 is an in-memory S3 that enforces conditional writes, so the backend's
// CAS mapping can be tested without a network or a real bucket.
type fakeS3 struct {
	obj     []byte
	etag    string
	getErr  error
	putErr  error
	lastPut *s3.PutObjectInput
}

func (f *fakeS3) GetObject(ctx context.Context, in *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.obj == nil {
		return nil, &smithy.GenericAPIError{Code: "NoSuchKey", Message: "missing"}
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(f.obj)), ETag: aws.String(f.etag)}, nil
}

func (f *fakeS3) PutObject(ctx context.Context, in *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	f.lastPut = in
	if f.putErr != nil {
		return nil, f.putErr
	}
	body, _ := io.ReadAll(in.Body)
	f.obj = body
	f.etag = `"w` + string(rune('0'+len(body)%10)) + `"`
	return &s3.PutObjectOutput{ETag: aws.String(f.etag)}, nil
}

func s3b(api s3API) S3Backend { return S3Backend{API: api, Bucket: "b", Key: "k", Label: "S3 b/k"} }

func TestS3GetMissingIsNil(t *testing.T) {
	blob, ver, err := s3b(&fakeS3{}).Get(context.Background())
	require.NoError(t, err)
	require.Nil(t, blob)
	require.Empty(t, ver)
}

func TestS3PutGetRoundTrip(t *testing.T) {
	f := &fakeS3{}
	b := s3b(f)
	require.NoError(t, b.Put(context.Background(), []byte("hello"), ""))
	blob, ver, err := b.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), blob)
	require.NotEmpty(t, ver)
}

func TestS3PutFirstUsesIfNoneMatch(t *testing.T) {
	f := &fakeS3{}
	require.NoError(t, s3b(f).Put(context.Background(), []byte("x"), ""))
	require.NotNil(t, f.lastPut.IfNoneMatch)
	require.Equal(t, "*", *f.lastPut.IfNoneMatch)
	require.Nil(t, f.lastPut.IfMatch)
}

func TestS3PutUpdateUsesIfMatch(t *testing.T) {
	f := &fakeS3{}
	require.NoError(t, s3b(f).Put(context.Background(), []byte("x"), `"etag123"`))
	require.NotNil(t, f.lastPut.IfMatch)
	require.Equal(t, `"etag123"`, *f.lastPut.IfMatch)
	require.Nil(t, f.lastPut.IfNoneMatch)
}

func TestS3PutPreconditionFailedIsRemoteChanged(t *testing.T) {
	f := &fakeS3{putErr: &smithy.GenericAPIError{Code: "PreconditionFailed", Message: "412"}}
	err := s3b(f).Put(context.Background(), []byte("x"), `"etag"`)
	require.ErrorIs(t, err, ErrRemoteChanged)
}

func TestS3PutConditionalConflictIsRemoteChanged(t *testing.T) {
	f := &fakeS3{putErr: &smithy.GenericAPIError{Code: "ConditionalRequestConflict", Message: "409"}}
	err := s3b(f).Put(context.Background(), []byte("x"), "")
	require.ErrorIs(t, err, ErrRemoteChanged)
}

func TestS3GetOtherErrorPropagates(t *testing.T) {
	f := &fakeS3{getErr: &smithy.GenericAPIError{Code: "AccessDenied", Message: "nope"}}
	_, _, err := s3b(f).Get(context.Background())
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrRemoteChanged))
}

// Pull/Push work over the S3 backend exactly as over local.
func TestS3PullPushViaBackend(t *testing.T) {
	f := &fakeS3{}
	b := s3b(f)
	require.NoError(t, Push(context.Background(), b, "team", sampleEvents(), ""))
	got, ver, err := Pull(context.Background(), b, "team")
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.NotEmpty(t, ver)
}

func TestNewS3BackendBuilds(t *testing.T) {
	b, err := NewS3Backend(context.Background(), S3Config{
		Endpoint: "https://example.r2.cloudflarestorage.com", Region: "auto",
		Bucket: "inv", Key: "meridian/ipspace.enc", AccessKey: "AK", Secret: "SK",
	})
	require.NoError(t, err)
	require.Contains(t, b.Describe(), "inv")
}
