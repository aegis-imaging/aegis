package storage

import (
	"context"
	"io"
	"time"
)

type Storage interface {
	GenerateUploadURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	GenerateDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	Store(ctx context.Context, key string, r io.Reader) error
	Retrieve(ctx context.Context, key string) (io.ReadCloser, error)
	List(ctx context.Context, prefix string) ([]string, error)
	Size(ctx context.Context, key string) (int64, error)
	Delete(ctx context.Context, key string) error
	Move(ctx context.Context, srcKey, dstKey string) error
	KeyToPath(key string) string
}
