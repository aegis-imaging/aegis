package storage

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	gcsapi "cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

type GCS struct {
	client *gcsapi.Client
	bucket string
}

func NewGCS(ctx context.Context, bucket string) (*GCS, error) {
	if bucket == "" {
		return nil, errors.New("GCS_BUCKET is required when STORAGE_MODE=gcs")
	}
	client, err := gcsapi.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create gcs client: %w", err)
	}
	return &GCS{client: client, bucket: bucket}, nil
}

func (g *GCS) signingOptions(key, method string, expiry time.Duration) (*gcsapi.SignedURLOptions, error) {
	email := os.Getenv("GCS_SIGNING_EMAIL")
	pk := parsePrivateKey(os.Getenv("GCS_SIGNING_PRIVATE_KEY"))
	if email == "" || len(pk) == 0 {
		return nil, errors.New("GCS signed URL configuration missing: set GCS_SIGNING_EMAIL and GCS_SIGNING_PRIVATE_KEY")
	}
	return &gcsapi.SignedURLOptions{
		GoogleAccessID: email,
		PrivateKey:     pk,
		Method:         method,
		Expires:        time.Now().Add(expiry),
		Scheme:         gcsapi.SigningSchemeV4,
	}, nil
}

func (g *GCS) GenerateUploadURL(_ context.Context, key string, expiry time.Duration) (string, error) {
	opts, err := g.signingOptions(key, "PUT", expiry)
	if err != nil {
		return "", err
	}
	opts.ContentType = "application/dicom"
	url, err := gcsapi.SignedURL(g.bucket, key, opts)
	if err != nil {
		return "", fmt.Errorf("sign gcs upload url: %w", err)
	}
	return url, nil
}

func (g *GCS) GenerateDownloadURL(_ context.Context, key string, expiry time.Duration) (string, error) {
	opts, err := g.signingOptions(key, "GET", expiry)
	if err != nil {
		return "", err
	}
	url, err := gcsapi.SignedURL(g.bucket, key, opts)
	if err != nil {
		return "", fmt.Errorf("sign gcs download url: %w", err)
	}
	return url, nil
}

func (g *GCS) Store(ctx context.Context, key string, r io.Reader) error {
	w := g.client.Bucket(g.bucket).Object(key).NewWriter(ctx)
	w.ContentType = "application/dicom"
	if _, err := io.Copy(w, r); err != nil {
		_ = w.CloseWithError(err)
		return err
	}
	return w.Close()
}

func (g *GCS) Retrieve(ctx context.Context, key string) (io.ReadCloser, error) {
	return g.client.Bucket(g.bucket).Object(key).NewReader(ctx)
}

func (g *GCS) List(ctx context.Context, prefix string) ([]string, error) {
	it := g.client.Bucket(g.bucket).Objects(ctx, &gcsapi.Query{Prefix: prefix + "/"})
	var keys []string
	for {
		attrs, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, err
		}
		if attrs != nil && attrs.Name != "" {
			keys = append(keys, attrs.Name)
		}
	}
	return keys, nil
}

func (g *GCS) Delete(ctx context.Context, key string) error {
	return g.client.Bucket(g.bucket).Object(key).Delete(ctx)
}

func (g *GCS) Move(ctx context.Context, srcKey, dstKey string) error {
	src := g.client.Bucket(g.bucket).Object(srcKey)
	dst := g.client.Bucket(g.bucket).Object(dstKey)
	if _, err := dst.CopierFrom(src).Run(ctx); err != nil {
		return err
	}
	return src.Delete(ctx)
}

func (g *GCS) KeyToPath(_ string) string {
	return ""
}

func parsePrivateKey(raw string) []byte {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	replaced := strings.ReplaceAll(raw, `\n`, "\n")
	if strings.Contains(replaced, "BEGIN PRIVATE KEY") {
		return []byte(replaced)
	}

	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err == nil && len(decoded) > 0 {
		return decoded
	}

	return []byte(replaced)
}
