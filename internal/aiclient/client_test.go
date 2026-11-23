package aiclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignatureMatchesPythonWireVector(t *testing.T) {
	body := []byte(`{"user_id":"u1","query":"find this","limit":10}`)
	got := signature("test-secret-with-at-least-32-bytes", "1730000000", "POST", searchPath, body)
	const want = "e834022df55ea511f9142326a0429c7309e7eae6667c6b31ca0d6ee358b753ba"
	if got != want {
		t.Fatalf("signature = %s, want %s", got, want)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestSearchSignsRequestAndDecodesResponse(t *testing.T) {
	secret := "test-secret-with-at-least-32-bytes"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != searchPath || r.Method != http.MethodPost {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request: %v", err)
		}
		var payload searchRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if payload.UserID != "user-1" || payload.ContentType != "application/pdf" {
			t.Errorf("unexpected signed request payload: %+v", payload)
		}
		timestamp := r.Header.Get("X-CloudVault-Timestamp")
		if got, want := r.Header.Get("X-CloudVault-Signature"), signature(secret, timestamp, r.Method, r.URL.Path, body); got != want {
			t.Errorf("signature = %s, want %s", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"file_id":"f1","file_name":"notes.txt","chunk_index":0,"page_number":null,"text":"hello world","score":0.9}]}`))
	})

	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, r)
			return rec.Result(), nil
		}),
	}

	result, err := NewWithClient("http://ai-service.local", secret, httpClient).Search(context.Background(), "user-1", "hello", 5, "application/pdf")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Results) != 1 || result.Results[0].FileID != "f1" {
		t.Fatalf("unexpected search results: %+v", result)
	}
}

func TestSearchRejectsMissingConfiguration(t *testing.T) {
	if _, err := New("", "").Search(context.Background(), "user-1", "hello", 5, ""); err != ErrNotConfigured {
		t.Fatalf("error = %v, want ErrNotConfigured", err)
	}
}
