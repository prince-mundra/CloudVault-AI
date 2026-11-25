package aiclient

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	searchPath = "/internal/v1/search"
	chatPath   = "/internal/v1/chat"
)

var ErrNotConfigured = errors.New("AI service is not configured")

type Client struct {
	baseURL string
	secret  string
	http    *http.Client
}

type SearchHit struct {
	FileID     string  `json:"file_id"`
	FileName   string  `json:"file_name"`
	ChunkIndex int     `json:"chunk_index"`
	PageNumber *int    `json:"page_number"`
	Text       string  `json:"text"`
	Score      float64 `json:"score"`
}

type SearchResponse struct {
	Results []SearchHit `json:"results"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	UserID  string        `json:"user_id"`
	Message string        `json:"message"`
	History []ChatMessage `json:"history"`
	FileIDs []string      `json:"file_ids,omitempty"`
}

type Citation struct {
	SourceNumber int     `json:"source_number"`
	FileID       string  `json:"file_id"`
	FileName     string  `json:"file_name"`
	ChunkIndex   int     `json:"chunk_index"`
	PageNumber   *int    `json:"page_number"`
	ChunkText    string  `json:"chunk_text"`
	Score        float64 `json:"score"`
}

type ChatResponse struct {
	Answer    string     `json:"answer"`
	Citations []Citation `json:"citations"`
}

type searchRequest struct {
	UserID      string `json:"user_id"`
	Query       string `json:"query"`
	Limit       int    `json:"limit"`
	ContentType string `json:"content_type,omitempty"`
}

func NewWithClient(baseURL, secret string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 70 * time.Second}
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), secret: secret, http: httpClient}
}

func New(baseURL, secret string) *Client {
	return NewWithClient(baseURL, secret, nil)
}

func (c *Client) Search(ctx context.Context, userID, query string, limit int, contentType string) (*SearchResponse, error) {
	if len(c.secret) < 32 || c.baseURL == "" {
		return nil, ErrNotConfigured
	}
	u, err := url.Parse(c.baseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return nil, ErrNotConfigured
	}
	payload, err := json.Marshal(searchRequest{UserID: userID, Query: query, Limit: limit, ContentType: contentType})
	if err != nil {
		return nil, fmt.Errorf("encoding AI query: %w", err)
	}
	endpoint := c.baseURL + searchPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("building AI request: %w", err)
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CloudVault-Timestamp", timestamp)
	req.Header.Set("X-CloudVault-Signature", signature(c.secret, timestamp, http.MethodPost, searchPath, payload))
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling AI service: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI service returned status %d", resp.StatusCode)
	}
	var result SearchResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding AI response: %w", err)
	}
	return &result, nil
}

func (c *Client) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if len(c.secret) < 32 || c.baseURL == "" {
		return nil, ErrNotConfigured
	}
	u, err := url.Parse(c.baseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return nil, ErrNotConfigured
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding AI chat request: %w", err)
	}
	endpoint := c.baseURL + chatPath
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("building AI chat request: %w", err)
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-CloudVault-Timestamp", timestamp)
	httpReq.Header.Set("X-CloudVault-Signature", signature(c.secret, timestamp, http.MethodPost, chatPath, payload))
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("calling AI chat service: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI chat service returned status %d", resp.StatusCode)
	}
	var result ChatResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding AI chat response: %w", err)
	}
	return &result, nil
}

func signature(secret, timestamp, method, path string, body []byte) string {
	bodyHash := sha256.Sum256(body)
	message := timestamp + "\n" + strings.ToUpper(method) + "\n" + path + "\n" + hex.EncodeToString(bodyHash[:])
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}
