package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

func inspectUpload(ctx context.Context, header *multipart.FileHeader, maxBytes int64) (string, string, error) {
	if header == nil {
		return "", "", fmt.Errorf("file is required")
	}
	if header.Size < 0 || header.Size > maxBytes {
		return "", "", fmt.Errorf("file exceeds configured upload limit")
	}
	f, err := header.Open()
	if err != nil {
		return "", "", fmt.Errorf("opening uploaded file: %w", err)
	}
	defer f.Close()
	sample := make([]byte, 512)
	n, readErr := io.ReadFull(f, sample)
	if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
		return "", "", fmt.Errorf("reading file signature: %w", readErr)
	}
	contentType := http.DetectContentType(sample[:n])
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", "", fmt.Errorf("rewinding uploaded file: %w", err)
	}
	h := sha256.New()
	written, err := io.Copy(h, &requestContextReader{ctx: ctx, reader: f})
	if err != nil {
		return "", "", fmt.Errorf("hashing uploaded file: %w", err)
	}
	if written > maxBytes {
		return "", "", fmt.Errorf("file exceeds configured upload limit")
	}
	if written != header.Size {
		return "", "", fmt.Errorf("uploaded file size does not match multipart metadata")
	}
	return hex.EncodeToString(h.Sum(nil)), contentType, nil
}

type requestContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *requestContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
