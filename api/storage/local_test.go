package storage

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocal_StoreRetrieve(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir, "http://localhost:8080")
	ctx := context.Background()

	err := s.Store(ctx, "test/file.dcm", strings.NewReader("DICOM-DATA"))
	require.NoError(t, err)

	rc, err := s.Retrieve(ctx, "test/file.dcm")
	require.NoError(t, err)
	defer rc.Close()

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "DICOM-DATA", string(data))
}

func TestLocal_List(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir, "http://localhost:8080")
	ctx := context.Background()

	require.NoError(t, s.Store(ctx, "prefix/a.dcm", strings.NewReader("a")))
	require.NoError(t, s.Store(ctx, "prefix/b.dcm", strings.NewReader("b")))

	keys, err := s.List(ctx, "prefix")
	require.NoError(t, err)
	assert.Len(t, keys, 2)
}

func TestLocal_ListEmpty(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir, "http://localhost:8080")

	keys, err := s.List(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, keys)
}

func TestLocal_Delete(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir, "http://localhost:8080")
	ctx := context.Background()

	require.NoError(t, s.Store(ctx, "test/del.dcm", strings.NewReader("data")))
	require.NoError(t, s.Delete(ctx, "test/del.dcm"))

	_, err := s.Retrieve(ctx, "test/del.dcm")
	assert.Error(t, err)
}

func TestLocal_Move(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir, "http://localhost:8080")
	ctx := context.Background()

	require.NoError(t, s.Store(ctx, "src/file.dcm", strings.NewReader("moved")))
	require.NoError(t, s.Move(ctx, "src/file.dcm", "dst/file.dcm"))

	rc, err := s.Retrieve(ctx, "dst/file.dcm")
	require.NoError(t, err)
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	assert.Equal(t, "moved", string(data))

	// Source should not exist
	_, err = s.Retrieve(ctx, "src/file.dcm")
	assert.Error(t, err)
}

func TestLocal_GenerateUploadURL(t *testing.T) {
	s := NewLocal("/data", "http://localhost:8080")
	url, err := s.GenerateUploadURL(context.Background(), "session/0", 0)
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/api/upload/file/session/0", url)
}

func TestLocal_GenerateDownloadURL(t *testing.T) {
	s := NewLocal("/data", "http://localhost:8080")
	url, err := s.GenerateDownloadURL(context.Background(), "dicom/raw/1.2.3/0.dcm", 0)
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/api/storage/dicom/raw/1.2.3/0.dcm", url)
}

func TestLocal_KeyToPath(t *testing.T) {
	s := NewLocal("/data", "http://localhost:8080")
	assert.Equal(t, "/data/test/file.dcm", s.KeyToPath("test/file.dcm"))
}

func TestLocal_KeyToPath_BlocksTraversal(t *testing.T) {
	s := NewLocal("/data", "http://localhost:8080")
	assert.Equal(t, "", s.KeyToPath("../../etc/passwd"))
	assert.Equal(t, "", s.KeyToPath("test/../../etc/passwd"))
}
