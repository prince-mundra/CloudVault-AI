package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFileJSONOmitsStorageLocation(t *testing.T) {
	file := File{ID: "file-id", Name: "report.pdf", Path: "private/owner/object", Url: "https://bucket.invalid/private", State: "active", SHA256: "abc"}
	data, err := json.Marshal(file)
	if err != nil {
		t.Fatal(err)
	}
	jsonText := string(data)
	for _, secret := range []string{"private/owner/object", "bucket.invalid", "\"path\"", "\"url\"", "\"state\""} {
		if strings.Contains(jsonText, secret) {
			t.Fatalf("API JSON exposed %q: %s", secret, jsonText)
		}
	}
}
