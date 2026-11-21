package storage

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalStorageRejectsTraversal(t *testing.T) {
	store, err := NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"../escape", "user/../../escape", "/absolute"} {
		if _, err := store.objectPath(key); err == nil {
			t.Errorf("expected key %q to be rejected", key)
		}
	}
}

func TestLocalStorageUploadDownloadDelete(t *testing.T) {
	root := t.TempDir()
	store, err := NewLocalStorage(root)
	if err != nil {
		t.Fatal(err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("CloudVault local storage")
	if _, err := part.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if err := request.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	header := request.MultipartForm.File["file"][0]

	if _, err := store.UploadFile(context.Background(), header, "user-a/file-1.txt"); err != nil {
		t.Fatal(err)
	}
	gotPath := filepath.Join(root, "user-a", "file-1.txt")
	if _, err := os.Stat(gotPath); err != nil {
		t.Fatalf("uploaded object missing: %v", err)
	}
	reader, err := store.DownloadFile(context.Background(), "user-a/file-1.txt")
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("downloaded %q, want %q", got, payload)
	}
	if err := store.DeleteFile(context.Background(), "user-a/file-1.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(gotPath); !os.IsNotExist(err) {
		t.Fatalf("expected object removed, stat err=%v", err)
	}
}
