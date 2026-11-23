package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/souvik03-136/Go-Store/internal/auth"
	"github.com/souvik03-136/Go-Store/internal/models"
	"github.com/souvik03-136/Go-Store/internal/repository"
	"github.com/souvik03-136/Go-Store/internal/services"
)

type testFileRepo struct {
	files map[string]*models.File
}

func (r *testFileRepo) GetFileByID(id string) (*models.File, error) {
	if f, ok := r.files[id]; ok {
		copy := *f
		return &copy, nil
	}
	return nil, errors.New("file not found")
}

func (r *testFileRepo) ListFilesByOwner(owner string) ([]*models.File, error) {
	var res []*models.File
	for _, f := range r.files {
		if f.OwnerID == owner {
			res = append(res, f)
		}
	}
	return res, nil
}

func (r *testFileRepo) SearchOwned(owner, search, ct, sort, order string, folderID *string, limit, offset int) ([]*models.File, error) {
	var res []*models.File
	for _, f := range r.files {
		if f.OwnerID == owner {
			res = append(res, f)
		}
	}
	return res, nil
}

func (r *testFileRepo) UpdateFile(f *models.File) error {
	r.files[f.ID] = f
	return nil
}

type testFolderRepo struct {
	folders map[string]*models.Folder
}

func (r *testFolderRepo) GetOwned(id, owner string) (*models.Folder, error) {
	if f, ok := r.folders[id]; ok && f.OwnerID == owner {
		return f, nil
	}
	return nil, errors.New("folder not found")
}

type testPermissionRepo struct{}

func (testPermissionRepo) GetPermission(string, string) (*models.Permission, error) {
	return nil, nil
}
func (testPermissionRepo) DeletePermission(string, string) error { return nil }

type testActivityRepo struct{}

func (testActivityRepo) Record(string, string, *string, string, string, int64) error { return nil }

type testUploadRepo struct {
	reservation *repository.UploadReservation
}

func (r *testUploadRepo) Reserve(_ context.Context, owner, hash, key string, size, quota int64) (*repository.UploadReservation, *models.File, error) {
	if r.reservation == nil {
		r.reservation = &repository.UploadReservation{
			ID:        "res-1",
			OwnerID:   owner,
			SHA256:    hash,
			ObjectKey: key,
			SizeBytes: size,
			ExpiresAt: time.Now().Add(time.Hour),
		}
	}
	return r.reservation, nil, nil
}

func (r *testUploadRepo) Expired(context.Context, string) ([]repository.UploadReservation, error) {
	return nil, nil
}
func (r *testUploadRepo) ReleaseExpired(context.Context, string) error { return nil }
func (r *testUploadRepo) Release(context.Context, string) error        { return nil }
func (r *testUploadRepo) Finalize(context.Context, string, *models.File, string) error {
	return nil
}
func (r *testUploadRepo) ExpiredAll(context.Context) ([]repository.UploadReservation, error) {
	return nil, nil
}

type testAuditRepo struct{}

func (testAuditRepo) Record(context.Context, repository.AuditEvent) error { return nil }

type testDeleteQueue struct{}

func (testDeleteQueue) Request(context.Context, *models.File, string) error { return nil }
func (testDeleteQueue) Claim(context.Context) (*repository.FileDeleteJob, error) {
	return nil, nil
}
func (testDeleteQueue) Complete(context.Context, *repository.FileDeleteJob) error { return nil }
func (testDeleteQueue) Fail(context.Context, *repository.FileDeleteJob, error) error {
	return nil
}

type testStorage struct {
	payload []byte
}

func (s *testStorage) UploadFile(context.Context, *multipart.FileHeader, string) (string, error) {
	return "test://object-key", nil
}
func (s *testStorage) DownloadFile(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(string(s.payload))), nil
}
func (s *testStorage) DeleteFile(context.Context, string) error { return nil }

func setupTestFileController() (*FileController, *testFileRepo) {
	gin.SetMode(gin.TestMode)
	fRepo := &testFileRepo{files: make(map[string]*models.File)}
	foRepo := &testFolderRepo{folders: make(map[string]*models.Folder)}
	store := &testStorage{payload: []byte("test-content")}

	svc := services.NewFileService(
		fRepo,
		foRepo,
		&testPermissionRepo{},
		&testActivityRepo{},
		&testUploadRepo{},
		&testAuditRepo{},
		&testDeleteQueue{},
		store,
		1024*1024*100, // 100 MB quota
		1024*1024*10,  // 10 MB upload limit
	)

	ctrl := NewFileController(svc, 1024*1024*10)
	return ctrl, fRepo
}

func TestFileController_UploadRequiresAuth(t *testing.T) {
	ctrl, _ := setupTestFileController()
	r := gin.New()
	r.POST("/files", ctrl.CreateFile)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/files", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
	}
}

func TestFileController_UploadValidFile(t *testing.T) {
	ctrl, _ := setupTestFileController()
	r := gin.New()
	r.POST("/files", func(c *gin.Context) {
		c.Set(auth.SubjectKey, "user-123")
		ctrl.CreateFile(c)
	})

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	part, err := writer.CreateFormFile("file", "sample.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("Hello world test content"))
	_ = writer.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/files", &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestFileController_GetFileOwnership(t *testing.T) {
	ctrl, fRepo := setupTestFileController()
	now := time.Now()
	fRepo.files["file-1"] = &models.File{
		ID:          "file-1",
		Name:        "doc.pdf",
		OwnerID:     "user-1",
		ContentType: "application/pdf",
		Size:        100,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	r := gin.New()
	r.GET("/files/:id", func(c *gin.Context) {
		// Mock requester as user-1 (owner)
		c.Set(auth.SubjectKey, "user-1")
		ctrl.GetFileByID(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/files/file-1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	// Now test unauthorized access from user-2
	r2 := gin.New()
	r2.GET("/files/:id", func(c *gin.Context) {
		c.Set(auth.SubjectKey, "user-2") // Different user
		ctrl.GetFileByID(c)
	})

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/files/file-1", nil)
	r2.ServeHTTP(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for cross-user file access, got %d", w2.Code)
	}
}

func TestFileController_DownloadFile(t *testing.T) {
	ctrl, fRepo := setupTestFileController()
	now := time.Now()
	fRepo.files["file-1"] = &models.File{
		ID:          "file-1",
		Name:        "report.pdf",
		OwnerID:     "user-1",
		ContentType: "application/pdf",
		Size:        12,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	r := gin.New()
	r.GET("/files/:id/download", func(c *gin.Context) {
		c.Set(auth.SubjectKey, "user-1")
		ctrl.DownloadFile(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/files/file-1/download", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Disposition"), "report.pdf") {
		t.Fatalf("expected Content-Disposition with report.pdf, got %s", w.Header().Get("Content-Disposition"))
	}
}

func TestFileController_DeleteFileOwnership(t *testing.T) {
	ctrl, fRepo := setupTestFileController()
	now := time.Now()
	fRepo.files["file-1"] = &models.File{
		ID:          "file-1",
		Name:        "secret.txt",
		OwnerID:     "user-1",
		ContentType: "text/plain",
		Size:        50,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	r := gin.New()
	r.DELETE("/files/:id", func(c *gin.Context) {
		// user-2 tries to delete user-1's file
		c.Set(auth.SubjectKey, "user-2")
		ctrl.DeleteFile(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/files/file-1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when deleting other user's file, got %d", w.Code)
	}

	// user-1 deletes own file
	r2 := gin.New()
	r2.DELETE("/files/:id", func(c *gin.Context) {
		c.Set(auth.SubjectKey, "user-1")
		ctrl.DeleteFile(c)
	})

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("DELETE", "/files/file-1", nil)
	r2.ServeHTTP(w2, req2)

	if w2.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted when deleting own file, got %d", w2.Code)
	}
}

func TestFileController_ListMyFiles(t *testing.T) {
	ctrl, fRepo := setupTestFileController()
	now := time.Now()
	fRepo.files["f1"] = &models.File{ID: "f1", Name: "a.txt", OwnerID: "user-1", CreatedAt: now}
	fRepo.files["f2"] = &models.File{ID: "f2", Name: "b.txt", OwnerID: "user-1", CreatedAt: now}
	fRepo.files["f3"] = &models.File{ID: "f3", Name: "c.txt", OwnerID: "user-other", CreatedAt: now}

	r := gin.New()
	r.GET("/files", func(c *gin.Context) {
		c.Set(auth.SubjectKey, "user-1")
		ctrl.ListMyFiles(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/files", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp struct {
		Data []*models.File `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 files for user-1, got %d", len(resp.Data))
	}
}
