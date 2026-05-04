package earnburn

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
		listProgramsFn: func(_ context.Context, customerID string) ([]EarnBurnProgram, error) {
			return []EarnBurnProgram{
				{CustomerSisfiID: "cs-1", CustomerID: customerID, Name: "Points", PointsRatio: 15, Active: true},
			}, nil
		},
	}
	r := setupAPIRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/programs?customer_id=cust-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var programs []EarnBurnProgram
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &programs))
	assert.Len(t, programs, 1)
	assert.Equal(t, "Points", programs[0].Name)
}

func TestCreateReward_API(t *testing.T) {
	repo := &mockRepo{
		createRewardAdminFn: func(_ context.Context, _ string, r *Reward) error {
			r.ID = "new-rw"
			return nil
		},
	}
	r := setupAPIRouter(repo)

	body := `{"name":"Cafe","points_cost":100}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/programs/cs-1/rewards", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, 201, w.Code)

	var resp Reward
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "new-rw", resp.ID)
	assert.Equal(t, "Cafe", resp.Name)
	assert.Equal(t, 100, resp.PointsCost)
	assert.Equal(t, "cs-1", resp.CustomerSisfiID)
	assert.True(t, resp.Active)
}

func TestListRewards_API(t *testing.T) {
	repo := &mockRepo{
		listAllRewardsFn: func(_ context.Context, customerSisfiID string) ([]Reward, error) {
			return []Reward{
				{ID: "rw-1", Name: "Cafe", PointsCost: 100},
				{ID: "rw-2", Name: "Pizza", PointsCost: 200},
			}, nil
		},
	}
	r := setupAPIRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/programs/cs-1/rewards", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var rewards []Reward
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rewards))
	assert.Len(t, rewards, 2)
}

func TestGetClientBalance_API(t *testing.T) {
	repo := &mockRepo{
		getBalanceFn: func(_ context.Context, clientID, customerSisfiID string) (int, error) {
			return 150, nil
		},
	}
	r := setupAPIRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/programs/cs-1/clients/client-1/balance", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(150), resp["balance"])
	assert.Equal(t, "client-1", resp["client_id"])
	assert.Equal(t, "cs-1", resp["customer_sisfi_id"])
}

func TestGetClientTransactions_API(t *testing.T) {
	repo := &mockRepo{
		listTransactionsFn: func(_ context.Context, _, _ string, _ int) ([]Transaction, error) {
			return []Transaction{
				{ID: "tx-1", Type: "earn", Amount: 10},
				{ID: "tx-2", Type: "burn", Amount: -5},
			}, nil
		},
	}
	r := setupAPIRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/programs/cs-1/clients/client-1/transactions", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var txs []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &txs))
	assert.Len(t, txs, 2)
}

func TestCreateCustomer_API(t *testing.T) {
	repo := &mockRepo{
		createCustomerFn: func(_ context.Context, c *Customer) error {
			c.ID = "new-cust"
			return nil
		},
	}
	r := setupAPIRouter(repo)

	body := `{"name":"Coffee Shop","slug":"coffee","phone":"+1234"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/customers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, 201, w.Code)
}

func TestGetCustomer_API(t *testing.T) {
	repo := &mockRepo{
		getCustomerFn: func(_ context.Context, id string) (*Customer, error) {
			return &Customer{ID: id, Name: "Coffee Shop", Slug: "coffee"}, nil
		},
	}
	r := setupAPIRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/customers/cust-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var cust Customer
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cust))
	assert.Equal(t, "Coffee Shop", cust.Name)
}

func TestCreateCollaborator_API(t *testing.T) {
	repo := &mockRepo{
		createCollaboratorFn: func(_ context.Context, c *Collaborator) error {
			c.ID = "new-collab"
			return nil
		},
	}
	r := setupAPIRouter(repo)

	body := `{"name":"Juan","phone":"+1234","hash_id":"abc123"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/customers/cust-1/collaborators", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, 201, w.Code)
}

func TestListCollaborators_API(t *testing.T) {
	repo := &mockRepo{
		listCollaboratorsFn: func(_ context.Context, _ string) ([]Collaborator, error) {
			return []Collaborator{{ID: "c-1", Name: "Juan"}}, nil
		},
	}
	r := setupAPIRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/customers/cust-1/collaborators", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestListClients_API(t *testing.T) {
	repo := &mockRepo{
		listClientsFn: func(_ context.Context, _ string) ([]Client, error) {
			return []Client{{ID: "cl-1", Name: "Maria"}}, nil
		},
	}
	r := setupAPIRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/customers/cust-1/clients", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestListFeedback_API(t *testing.T) {
	repo := &mockRepo{
		listFeedbackFn: func(_ context.Context, _ string) ([]FeedbackEntry, error) {
			return []FeedbackEntry{{ID: "f-1", Message: "Great!"}}, nil
		},
	}
	r := setupAPIRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/customers/cust-1/feedback", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestUpdateReward_API(t *testing.T) {
	r := setupAPIRouter(&mockRepo{})

	body := `{"name":"Updated","points_cost":200}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/programs/cs-1/rewards/rw-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}
