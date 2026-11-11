package ipspace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithy "github.com/aws/smithy-go"
)

// s3API is the slice of the S3 client the backend uses; *s3.Client satisfies it,
// and tests inject a fake so the engine stays offline-testable.
type s3API interface {
	GetObject(ctx context.Context, in *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, in *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, in *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

// S3Backend stores the shared blob as one object in an S3-compatible bucket
// (AWS S3, Cloudflare R2, Backblaze B2, MinIO…). Its version token is the
// object's ETag, and it uses S3 conditional writes (If-None-Match on the first
// push, If-Match on an update) for a true compare-and-swap, so a concurrent push
// is refused by the service rather than silently clobbered. Requires a provider
// that honors conditional writes (AWS S3 and R2 do).
type S3Backend struct {
	API    s3API
	Bucket string
	Key    string
	Label  string
}

// S3Config is the user-supplied connection detail for an S3-compatible bucket.
type S3Config struct {
	Endpoint     string // custom endpoint for S3-compatible services; empty for AWS S3
	Region       string
	Bucket       string
	Key          string // object key, e.g. "meridian/ipspace.enc"
	AccessKey    string
	Secret       string
	UsePathStyle bool
}

// NewS3Backend builds a backend from config. Static credentials are used when
// provided; otherwise the AWS default credential chain applies. A custom
// endpoint turns on path-style addressing (what MinIO and most compatibles want).
func NewS3Backend(ctx context.Context, c S3Config) (*S3Backend, error) {
	if c.Bucket == "" || c.Key == "" {
		return nil, fmt.Errorf("S3 needs a bucket and an object key")
	}
	region := c.Region
	if region == "" {
		region = "us-east-1"
	}
	loadOpts := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(region)}
	if c.AccessKey != "" {
		loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(c.AccessKey, c.Secret, "")))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if c.Endpoint != "" {
			o.BaseEndpoint = aws.String(c.Endpoint)
		}
		if c.UsePathStyle || c.Endpoint != "" {
			o.UsePathStyle = true
		}
	})
	return &S3Backend{API: client, Bucket: c.Bucket, Key: c.Key, Label: c.Describe()}, nil
}

// Describe is a short, credential-free location label for the UI.
func (c S3Config) Describe() string {
	label := fmt.Sprintf("S3 %s/%s", c.Bucket, c.Key)
	if c.Endpoint != "" {
		label += " @ " + c.Endpoint
	}
	return label
}

// VerifyConditionalWrites confirms the bucket actually honors conditional writes
// — the primitive the whole lost-update guard rests on. It writes and deletes a
// throwaway probe object (never the real store key): a provider that rejects the
// If-None-Match header (e.g. Backblaze B2) or silently ignores it (older MinIO)
// is detected and rejected here, at setup, rather than losing a teammate's push
// later. Compliant providers (AWS S3, Cloudflare R2, modern MinIO) pass.
func (b S3Backend) VerifyConditionalWrites(ctx context.Context) error {
	probeKey := b.Key + ".meridian-cas-probe"
	put := func(ifNoneMatch bool) error {
		in := &s3.PutObjectInput{Bucket: &b.Bucket, Key: &probeKey, Body: bytes.NewReader([]byte("probe"))}
		if ifNoneMatch {
			in.IfNoneMatch = aws.String("*")
		}
		_, err := b.API.PutObject(ctx, in)
		return err
	}
	cleanup := func() {
		_, _ = b.API.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &b.Bucket, Key: &probeKey})
	}

	// 1. Create the probe with If-None-Match:*. If a leftover probe exists from a
	//    crashed prior run, that's a 412 — delete and retry once.
	if err := put(true); err != nil {
		if isS3PreconditionFailed(err) {
			cleanup()
			err = put(true)
		}
		if err != nil {
			return fmt.Errorf("this storage provider doesn't support the conditional writes Meridian needs to sync safely (so two people can't overwrite each other): %w", err)
		}
	}
	defer cleanup()

	// 2. A second If-None-Match:* MUST be refused now that the object exists. If it
	//    succeeds, the provider ignores conditional headers and can't protect
	//    against lost updates.
	err := put(true)
	if err == nil {
		return fmt.Errorf("this storage provider ignores conditional writes, so Meridian can't stop two people from overwriting each other's changes — it won't sync here safely")
	}
	if !isS3PreconditionFailed(err) {
		return fmt.Errorf("couldn't verify conditional-write support on this bucket: %w", err)
	}
	return nil
}

// Get reads the object and returns its bytes and ETag, or (nil, "", nil) when the
// object does not exist yet.
func (b S3Backend) Get(ctx context.Context) ([]byte, string, error) {
	out, err := b.API.GetObject(ctx, &s3.GetObjectInput{Bucket: &b.Bucket, Key: &b.Key})
	if err != nil {
		if isS3NotFound(err) {
			return nil, "", nil
		}
		return nil, "", err
	}
	defer out.Body.Close()
	blob, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, "", err
	}
	ver := ""
	if out.ETag != nil {
		ver = *out.ETag
	}
	return blob, ver, nil
}

// Put writes the object with a conditional header so the remote is replaced only
// if it still holds expectedVersion ("" meaning the object must not exist yet).
// A precondition failure maps to ErrRemoteChanged.
func (b S3Backend) Put(ctx context.Context, blob []byte, expectedVersion string) error {
	in := &s3.PutObjectInput{Bucket: &b.Bucket, Key: &b.Key, Body: bytes.NewReader(blob)}
	if expectedVersion == "" {
		in.IfNoneMatch = aws.String("*")
	} else {
		in.IfMatch = aws.String(expectedVersion)
	}
	if _, err := b.API.PutObject(ctx, in); err != nil {
		if isS3PreconditionFailed(err) {
			return ErrRemoteChanged
		}
		return err
	}
	return nil
}

// Verify confirms the bucket honors conditional writes (reaching it validates
// the endpoint and credentials too).
func (b S3Backend) Verify(ctx context.Context) error { return b.VerifyConditionalWrites(ctx) }

// Describe returns the bucket/key (and endpoint) for the UI.
func (b S3Backend) Describe() string { return b.Label }

func isS3NotFound(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NoSuchKey", "NotFound":
			return true
		}
	}
	return false
}

func isS3PreconditionFailed(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "PreconditionFailed", "ConditionalRequestConflict":
			return true
		}
	}
	return false
}
