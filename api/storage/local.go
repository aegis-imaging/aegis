package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Local struct {
	baseDir string
	apiBase string
}

func NewLocal(baseDir, apiBase string) *Local {
	return &Local{baseDir: baseDir, apiBase: apiBase}
}

func (l *Local) GenerateUploadURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return fmt.Sprintf("%s/api/upload/file/%s", l.apiBase, key), nil
}

func (l *Local) GenerateDownloadURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return fmt.Sprintf("%s/api/storage/%s", l.apiBase, key), nil
}

func (l *Local) Store(_ context.Context, key string, r io.Reader) error {
	path := filepath.Join(l.baseDir, key)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func (l *Local) Retrieve(_ context.Context, key string) (io.ReadCloser, error) {
	path := filepath.Join(l.baseDir, key)
	return os.Open(path)
}

func (l *Local) List(_ context.Context, prefix string) ([]string, error) {
	dir := filepath.Join(l.baseDir, prefix)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var keys []string
	for _, e := range entries {
		if !e.IsDir() {
			keys = append(keys, filepath.Join(prefix, e.Name()))
		}
	}
	return keys, nil
}

func (l *Local) Delete(_ context.Context, key string) error {
	path := filepath.Join(l.baseDir, key)
	return os.Remove(path)
}

func (l *Local) Move(_ context.Context, srcKey, dstKey string) error {
	srcPath := filepath.Join(l.baseDir, srcKey)
	dstPath := filepath.Join(l.baseDir, dstKey)
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}
	if err := os.Rename(srcPath, dstPath); err != nil {
		return l.copyFile(srcPath, dstPath)
	}
	return nil
}

func (l *Local) copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return os.Remove(src)
}

func (l *Local) KeyToPath(key string) string {
	clean := filepath.Clean(key)
	if strings.Contains(clean, "..") {
		return ""
	}
	return filepath.Join(l.baseDir, clean)
}
