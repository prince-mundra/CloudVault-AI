package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

// LocalStorage persists objects beneath a configured root directory. The
// destination is treated as an untrusted relative key and may not escape root.
type LocalStorage struct {
	root string
}

func NewLocalStorage(root string) (*LocalStorage, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("local storage root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolving local storage root: %w", err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("creating local storage root: %w", err)
	}
	rootPath, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("resolving local storage root: %w", err)
	}
	return &LocalStorage{root: rootPath}, nil
}

func (s *LocalStorage) objectPath(key string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(key))
	if key == "" || filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid storage object key")
	}
	path := filepath.Join(s.root, clean)
	rel, err := filepath.Rel(s.root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid storage object key")
	}
	// Reject any existing symlink component. This closes the straightforward
	// symlink escape path in addition to the lexical traversal check above.
	current := s.root
	for _, component := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) {
			break
		}
		if statErr != nil {
			return "", fmt.Errorf("checking storage path: %w", statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("storage object path contains a symlink")
		}
	}
	return path, nil
}

func (s *LocalStorage) UploadFile(ctx context.Context, file *multipart.FileHeader, destination string) (string, error) {
	if file == nil {
		return "", fmt.Errorf("file is required")
	}
	path, err := s.objectPath(destination)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return "", fmt.Errorf("creating object directory: %w", err)
	}
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("opening uploaded file: %w", err)
	}
	defer src.Close()
	dst, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		return "", fmt.Errorf("creating local object: %w", err)
	}
	_, copyErr := io.Copy(dst, &contextReader{ctx: ctx, r: src})
	closeErr := dst.Close()
	if copyErr != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("writing local object: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("closing local object: %w", closeErr)
	}
	return "local://" + filepath.ToSlash(destination), nil
}

func (s *LocalStorage) DeleteFile(ctx context.Context, objectKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := s.objectPath(objectKey)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting local object: %w", err)
	}
	return nil
}

func (s *LocalStorage) DownloadFile(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path, err := s.objectPath(objectKey)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening local object: %w", err)
	}
	return f, nil
}

// contextReader lets long local writes stop promptly when the request is canceled.
type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}
