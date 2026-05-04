package cashback

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theluisbolivar/fidel-quick/internal/apperror"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupAPIRouter(repo *mockRepo) *gin.Engine {
	svc := NewService(repo, newMockCache(), testLogger())
	handler := NewAPIHandler(svc)

	r := gin.New()
	r.Use(apperror.ErrorHandler(testLogger()))
	v1 := r.Group("/api/v1")
	handler.RegisterRoutes(v1)
	return r
}

func TestListPrograms_API(t *testing.T) {
	repo := &mockRepo{
		listProgramsFn: func(_ context.Context, _ string) ([]CashbackProgram, error) {
			return []CashbackProgram{
				{CustomerSisfiID: "cs-1", Name: "Cashback 5%", CashbackRate: 0.05},
			}, nil
		},
	}
	r := setupAPIRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/cashback-programs?customer_id=cust-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var programs []CashbackProgram
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &programs))
	assert.Len(t, programs, 1)
}

func TestCreateReward_API(t *testing.T) {
	repo := &mockRepo{
		createRewardAdminFn: func(_ context.Context, _ string, r *CashbackReward) error {
			r.ID = "new-rw"
			return nil
		},
	}
	r := setupAPIRouter(repo)

	body := `{"name":"Descuento $5","cost":5.0}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/customer-sisfi/cs-1/rewards", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, 201, w.Code)
}

func TestListRewards_API(t *testing.T) {
	repo := &mockRepo{
		listAllRewardsFn: func(_ context.Context, _ string) ([]CashbackReward, error) {
			return []CashbackReward{
				{ID: "rw-1", Name: "Descuento $5", Cost: 5.0},
			}, nil
		},
	}
	r := setupAPIRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/customer-sisfi/cs-1/rewards", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var rewards []CashbackReward
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rewards))
	assert.Len(t, rewards, 1)
}

func TestGetClientBalance_API(t *testing.T) {
	repo := &mockRepo{
		getBalanceFn: func(_ context.Context, _, _ string) (float64, error) {
			return 25.50, nil
		},
	}
	r := setupAPIRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/customer-sisfi/cs-1/clients/client-1/balance", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 25.50, resp["balance"])
}

func TestGetClientTransactions_API(t *testing.T) {
	repo := &mockRepo{
		listTransactionsFn: func(_ context.Context, _, _ string, _ int) ([]CashbackTransaction, error) {
			return []CashbackTransaction{
				{ID: "tx-1", Type: "earn", Amount: 5.0},
			}, nil
		},
	}
	r := setupAPIRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/customer-sisfi/cs-1/clients/client-1/transactions", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var txs []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &txs))
	assert.Len(t, txs, 1)
}

func TestUpdateReward_API(t *testing.T) {
	r := setupAPIRouter(&mockRepo{})

	body := `{"name":"Updated","cost":10.0}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/customer-sisfi/cs-1/rewards/rw-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}
