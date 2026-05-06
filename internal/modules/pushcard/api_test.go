package pushcard

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

func newAPIRouter(t *testing.T) (*gin.Engine, *fakeRepo) {
	t.Helper()
	repo := newFakeRepo()
	svc := NewService(repo, &fakeCache{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	h := NewAPIHandler(svc)

	r := gin.New()
	r.Use(apperror.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))))
	v1 := r.Group("/api/v1")
	h.RegisterRoutes(v1)
	return r, repo
}

func TestAPI_GetConfigByCustomer_Missing(t *testing.T) {
	r, _ := newAPIRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/pushcard/config?customer_id=", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for missing customer_id, got %d", w.Code)
	}
}

func TestAPI_UpsertConfig_RejectsZeroSlots(t *testing.T) {
	r, _ := newAPIRouter(t)

	body := strings.NewReader(`{"card_slots": 0}`)
	req, _ := http.NewRequest("PUT", "/api/v1/pushcard/programs/cs-1/config", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for slots=0, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestAPI_UpsertConfig_OK(t *testing.T) {
	r, repo := newAPIRouter(t)

	body := strings.NewReader(`{"card_slots": 8}`)
	req, _ := http.NewRequest("PUT", "/api/v1/pushcard/programs/cs-1/config", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if cfg, ok := repo.configs["cs-1"]; !ok || cfg.CardSlots != 8 {
		t.Fatalf("config not persisted; got %+v", repo.configs)
	}
}

func TestAPI_CreateProgram_OK(t *testing.T) {
	r, repo := newAPIRouter(t)

	body := strings.NewReader(`{"customer_id":"c1","card_slots":12}`)
	req, _ := http.NewRequest("POST", "/api/v1/pushcard/programs", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d, body=%s", w.Code, w.Body.String())
	}
	var got Config
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.CustomerID != "c1" || got.CardSlots != 12 || got.CustomerSisfiID == "" {
		t.Fatalf("unexpected response: %+v", got)
	}
	if _, ok := repo.configs[got.CustomerSisfiID]; !ok {
		t.Fatalf("config not persisted")
	}
}

func TestAPI_CreateProgram_RejectsInvalidSlots(t *testing.T) {
	r, _ := newAPIRouter(t)

	cases := []string{
		`{"customer_id":"c1","card_slots":0}`,
		`{"customer_id":"c1","card_slots":51}`,
	}
	for _, body := range cases {
		req, _ := http.NewRequest("POST", "/api/v1/pushcard/programs", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("want 400 for body=%s, got %d (body=%s)", body, w.Code, w.Body.String())
		}
	}
}

func TestAPI_CreateProgram_Conflict(t *testing.T) {
	r, _ := newAPIRouter(t)

	// First create — should succeed.
	first := strings.NewReader(`{"customer_id":"c1","card_slots":10}`)
	req1, _ := http.NewRequest("POST", "/api/v1/pushcard/programs", first)
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("first create want 201, got %d", w1.Code)
	}

	// Second create for same customer — should 409.
	second := strings.NewReader(`{"customer_id":"c1","card_slots":10}`)
	req2, _ := http.NewRequest("POST", "/api/v1/pushcard/programs", second)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("second create want 409, got %d, body=%s", w2.Code, w2.Body.String())
	}
}

func TestAPI_ListCards_Empty(t *testing.T) {
	r, _ := newAPIRouter(t)

	req, _ := http.NewRequest("GET", "/api/v1/pushcard/programs/cs-1/cards", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var cards []Card
	if err := json.Unmarshal(w.Body.Bytes(), &cards); err != nil && len(w.Body.Bytes()) > 0 && string(w.Body.Bytes()) != "null" {
		t.Fatalf("decode: %v", err)
	}
	if len(cards) != 0 {
		t.Fatalf("expected empty, got %d", len(cards))
	}
}
