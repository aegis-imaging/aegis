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
	iamcredentials "google.golang.org/api/iamcredentials/v1"
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

// signingOptions builds SignedURLOptions using either an explicit private key
// (GCS_SIGNING_PRIVATE_KEY) or IAM-based signing (recommended on Cloud Run).
// Only GCS_SIGNING_EMAIL is required; the private key is optional.
func (g *GCS) signingOptions(ctx context.Context, method string, expiry time.Duration) (*gcsapi.SignedURLOptions, error) {
	email := os.Getenv("GCS_SIGNING_EMAIL")
	if email == "" {
		return nil, errors.New("GCS_SIGNING_EMAIL is required for GCS signed URLs")
	}
	opts := &gcsapi.SignedURLOptions{
		GoogleAccessID: email,
		Method:         method,
		Expires:        time.Now().Add(expiry),
		Scheme:         gcsapi.SigningSchemeV4,
	}
	if pk := parsePrivateKey(os.Getenv("GCS_SIGNING_PRIVATE_KEY")); len(pk) > 0 {
		opts.PrivateKey = pk
	} else {
		// Fall back to IAM-based signing. The Cloud Run service account needs
		// roles/iam.serviceAccountTokenCreator on itself.
		opts.SignBytes = func(b []byte) ([]byte, error) {
			svc, err := iamcredentials.NewService(ctx)
			if err != nil {
				return nil, fmt.Errorf("iam credentials service: %w", err)
			}
			resource := "projects/-/serviceAccounts/" + email
			resp, err := svc.Projects.ServiceAccounts.SignBlob(resource,
				&iamcredentials.SignBlobRequest{Payload: base64.StdEncoding.EncodeToString(b)},
			).Context(ctx).Do()
			if err != nil {
				return nil, fmt.Errorf("iam sign blob: %w", err)
			}
			return base64.StdEncoding.DecodeString(resp.SignedBlob)
		}
	}
	return opts, nil
}

func (g *GCS) GenerateUploadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	opts, err := g.signingOptions(ctx, "PUT", expiry)
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

func (g *GCS) GenerateDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	opts, err := g.signingOptions(ctx, "GET", expiry)
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
