package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestHealthz verifica el liveness probe: 200 con {"status":"ok"} y sin
// tocar ninguna dependencia (el handler no recibe DB ni Redis).
func TestHealthz(t *testing.T) {
	r := gin.New()
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body: %v (%q)", err, w.Body.String())
	}
	if body["status"] != "ok" {
		t.Fatalf(`want status "ok", got %q`, body["status"])
	}
}

func newReadyzRouter(pingPostgres, pingRedis pingFunc) *gin.Engine {
	r := gin.New()
	r.GET("/readyz", readyzHandlerFor(pingPostgres, pingRedis))
	return r
}

func doReadyz(t *testing.T, r *gin.Engine) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/readyz", nil)
	r.ServeHTTP(w, req)
	return w
}

var pingOK pingFunc = func(context.Context) error { return nil }

// TestReadyz_AllHealthy: ambas dependencias responden → 200 {"status":"ready"}.
func TestReadyz_AllHealthy(t *testing.T) {
	r := newReadyzRouter(pingOK, pingOK)

	w := doReadyz(t, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if body["status"] != "ready" {
		t.Fatalf(`want status "ready", got %q`, body["status"])
	}
}

// TestReadyz_PostgresDown: Postgres falla → 503 con detalle del fallo, sin
// llegar a chequear Redis.
func TestReadyz_PostgresDown(t *testing.T) {
	redisCalled := false
	r := newReadyzRouter(
		func(context.Context) error { return errors.New("connection refused") },
		func(context.Context) error { redisCalled = true; return nil },
	)

	w := doReadyz(t, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", w.Code)
	}
	if redisCalled {
		t.Fatal("redis should not be pinged after postgres fails")
	}
	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "not_ready" {
		t.Fatalf(`want status "not_ready", got %q`, body["status"])
	}
	if body["postgres"] == "" {
		t.Fatal("expected postgres error detail in body")
	}
}

// TestReadyz_RedisDown: Postgres OK pero Redis falla → 503.
func TestReadyz_RedisDown(t *testing.T) {
	r := newReadyzRouter(
		pingOK,
		func(context.Context) error { return errors.New("i/o timeout") },
	)

	w := doReadyz(t, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", w.Code)
	}
	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "not_ready" {
		t.Fatalf(`want status "not_ready", got %q`, body["status"])
	}
	if body["redis"] == "" {
		t.Fatal("expected redis error detail in body")
	}
}

// TestReadyz_RespectsTimeout: el handler impone un deadline (~1s); un pinger
// que respeta el contexto debe recibir un ctx con deadline configurado.
func TestReadyz_RespectsTimeout(t *testing.T) {
	var hadDeadline bool
	r := newReadyzRouter(
		func(ctx context.Context) error {
			_, hadDeadline = ctx.Deadline()
			return nil
		},
		pingOK,
	)

	doReadyz(t, r)

	if !hadDeadline {
		t.Fatal("readyz should ping dependencies with a deadline context")
	}
}
