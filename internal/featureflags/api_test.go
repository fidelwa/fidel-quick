package featureflags

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/theluisbolivar/fidel-quick/internal/apperror"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newAPIRouter(repo *fakeRepo) *gin.Engine {
	svc := NewService(repo, newCountingCache(), discardLogger())
	h := NewAPIHandler(svc)

	r := gin.New()
	r.Use(apperror.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))))
	v1 := r.Group("/api/v1")
	h.RegisterRoutes(v1)
	return r
}

func TestAPI_ListFlags_Empty(t *testing.T) {
	r := newAPIRouter(newFakeRepo())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/flags", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatalf("want empty array, got %s", w.Body.String())
	}
}

func TestAPI_ListFlags_ReturnsFlags(t *testing.T) {
	repo := newFakeRepo()
	repo.flags["a"] = &Flag{Key: "a", EnabledGlobally: true, CustomerOverrides: map[string]bool{}}
	r := newAPIRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/flags", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var flags []Flag
	if err := json.Unmarshal(w.Body.Bytes(), &flags); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(flags) != 1 || flags[0].Key != "a" || !flags[0].EnabledGlobally {
		t.Fatalf("unexpected flags: %+v", flags)
	}
}

func TestAPI_UpdateFlag_CreatesAndToggles(t *testing.T) {
	repo := newFakeRepo()
	r := newAPIRouter(repo)

	body := strings.NewReader(`{"enabled_globally": true, "description": "beta"}`)
	req, _ := http.NewRequest("PUT", "/api/v1/admin/flags/admin.beta", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var flag Flag
	if err := json.Unmarshal(w.Body.Bytes(), &flag); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if flag.Key != "admin.beta" || !flag.EnabledGlobally || flag.Description != "beta" {
		t.Fatalf("unexpected flag: %+v", flag)
	}
	// Persisted (upsert created it).
	if _, ok := repo.flags["admin.beta"]; !ok {
		t.Fatal("flag was not persisted")
	}
}

func TestAPI_UpdateFlag_PartialLeavesOtherFieldsUnchanged(t *testing.T) {
	repo := newFakeRepo()
	repo.flags["f"] = &Flag{Key: "f", EnabledGlobally: true, DefaultValue: true, CustomerOverrides: map[string]bool{}}
	r := newAPIRouter(repo)

	// Only toggle enabled_globally off; default_value must stay true.
	body := strings.NewReader(`{"enabled_globally": false}`)
	req, _ := http.NewRequest("PUT", "/api/v1/admin/flags/f", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var flag Flag
	_ = json.Unmarshal(w.Body.Bytes(), &flag)
	if flag.EnabledGlobally {
		t.Fatal("enabled_globally should be false")
	}
	if !flag.DefaultValue {
		t.Fatal("default_value should have been preserved as true")
	}
}

func TestAPI_UpdateFlag_InvalidBody(t *testing.T) {
	r := newAPIRouter(newFakeRepo())

	body := strings.NewReader(`{not json`)
	req, _ := http.NewRequest("PUT", "/api/v1/admin/flags/f", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for invalid body, got %d", w.Code)
	}
}
