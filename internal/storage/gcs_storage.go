// internal/storage/gcs_storage.go

package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// GCSStorage implements Storage using Google Cloud Storage.
type GCSStorage struct {
	client *storage.Client
	bucket string
}

// NewGCSStorage initialises a GCS client using the credentials file at
// credentialsFile and returns a GCSStorage targeting the given bucket.
func NewGCSStorage(ctx context.Context, credentialsFile, bucket string) (*GCSStorage, error) {
	if credentialsFile == "" || bucket == "" {
		return nil, fmt.Errorf("credentialsFile and bucket are required")
	}

	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("creating GCS client: %w", err)
	}

	return &GCSStorage{client: client, bucket: bucket}, nil
}

// UploadFile uploads the given multipart file to GCS under the specified
// destination path. No public URL is returned.
func (g *GCSStorage) UploadFile(ctx context.Context, file *multipart.FileHeader, destination string) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("opening uploaded file: %w", err)
	}
	defer src.Close()

	obj := g.client.Bucket(g.bucket).Object(destination)
	wc := obj.NewWriter(ctx)
	wc.ContentType = file.Header.Get("Content-Type")

	if _, err := io.Copy(wc, src); err != nil {
		_ = wc.Close()
		return "", fmt.Errorf("copying file to GCS (bucket=%s object=%s): %w", g.bucket, destination, err)
	}

	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("closing GCS writer (bucket=%s object=%s): %w", g.bucket, destination, err)
	}

	return "", nil
}

// DeleteFile removes the object at objectKey from GCS.
func (g *GCSStorage) DeleteFile(ctx context.Context, objectKey string) error {
	if err := g.client.Bucket(g.bucket).Object(objectKey).Delete(ctx); err != nil && !errors.Is(err, storage.ErrObjectNotExist) {
		return fmt.Errorf("deleting from GCS (bucket=%s key=%s): %w", g.bucket, objectKey, err)
	}
	return nil
}

func (g *GCSStorage) DownloadFile(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	r, err := g.client.Bucket(g.bucket).Object(objectKey).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("downloading from GCS (bucket=%s object=%s): %w", g.bucket, objectKey, err)
	}
	return r, nil
}
