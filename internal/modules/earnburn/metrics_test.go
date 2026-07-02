package earnburn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theluisbolivar/fidel-quick/internal/apperror"
)

// seededRaw representa un customer con datos sembrados: 10 clientes registrados,
// 6 activos en la ventana, 2 reactivados, 5 participantes con 20 compras que
// suman $50,000 de gasto; cashback generó $2,500, redimió $900 (= costo) y
// mantiene $1,600 de saldo sin redimir; 8 canjes creados, 6 confirmados.
func seededRaw() MetricsRaw {
	return MetricsRaw{
		RegisteredClients:    10,
		ActiveClients:        6,
		ReactivatedClients:   2,
		ParticipatingClients: 5,
		PurchaseCount:        20,
		TotalSpend:           50000,
		BenefitsGenerated:    2500,
		BenefitsRedeemed:     900,
		BenefitsCost:         900,
		OutstandingValue:     1600,
		TotalRedemptions:     8,
		ConfirmedRedemptions: 6,
	}
}

func TestComputeMetrics_SeededData(t *testing.T) {
	m := ComputeMetrics(seededRaw())

	assert.Equal(t, 10, m.RegisteredClients)
	assert.Equal(t, 6, m.ActiveClients)
	assert.Equal(t, 2, m.ReactivatedClients)

	assert.InDelta(t, 0.50, m.ParticipationRate, 1e-9) // 5 / 10
	assert.InDelta(t, 4.0, m.PurchaseFrequency, 1e-9)  // 20 / 5
	assert.InDelta(t, 2500.0, m.AverageTicket, 1e-9)   // 50000 / 20
	assert.InDelta(t, 10000.0, m.SpendPerClient, 1e-9) // 50000 / 5
	assert.InDelta(t, 50000.0, m.TotalSpend, 1e-9)
	assert.InDelta(t, 2500.0, m.BenefitsGenerated, 1e-9)
	assert.InDelta(t, 900.0, m.BenefitsRedeemed, 1e-9)
	assert.InDelta(t, 900.0, m.BenefitsCost, 1e-9)
	assert.InDelta(t, 1600.0, m.PotentialGain, 1e-9)
	assert.InDelta(t, 0.75, m.RedemptionRate, 1e-9) // 6 / 8
}

func TestComputeMetrics_ZeroDivisionSafe(t *testing.T) {
	// Customer sin clientes ni actividad: nada debe dividir por cero ni producir NaN/Inf.
	m := ComputeMetrics(MetricsRaw{})

	assert.Equal(t, 0, m.RegisteredClients)
	assert.Equal(t, float64(0), m.ParticipationRate)
	assert.Equal(t, float64(0), m.PurchaseFrequency)
	assert.Equal(t, float64(0), m.AverageTicket)
	assert.Equal(t, float64(0), m.SpendPerClient)
	assert.Equal(t, float64(0), m.RedemptionRate)
}

func TestComputeMetrics_Rounding(t *testing.T) {
	// 1 participante / 3 registrados = 0.3333… → 0.33; 100 / 3 compras = 33.33.
	m := ComputeMetrics(MetricsRaw{
		RegisteredClients:    3,
		ParticipatingClients: 1,
		PurchaseCount:        3,
		TotalSpend:           100,
		TotalRedemptions:     3,
		ConfirmedRedemptions: 1,
	})
	assert.InDelta(t, 0.33, m.ParticipationRate, 1e-9)
	assert.InDelta(t, 3.0, m.PurchaseFrequency, 1e-9)
	assert.InDelta(t, 33.33, m.AverageTicket, 1e-9)
	assert.InDelta(t, 0.33, m.RedemptionRate, 1e-9)
}

func TestGetMetrics_API(t *testing.T) {
	repo := &mockRepo{
		getMetricsRawFn: func(_ context.Context, customerID string) (MetricsRaw, error) {
			assert.Equal(t, "cust-1", customerID)
			return seededRaw(), nil
		},
	}
	r := setupAPIRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/customers/cust-1/metrics", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var m CustomerMetrics
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &m))
	assert.Equal(t, 10, m.RegisteredClients)
	assert.InDelta(t, 0.50, m.ParticipationRate, 1e-9)
	assert.InDelta(t, 2500.0, m.AverageTicket, 1e-9)
	assert.InDelta(t, 0.75, m.RedemptionRate, 1e-9)
}

func TestGetMetrics_API_OwnershipMismatch(t *testing.T) {
	// Cuando el JWT fija un customer_id distinto al :id, el endpoint responde 404
	// sin llegar a consultar métricas.
	repo := &mockRepo{
		getMetricsRawFn: func(_ context.Context, _ string) (MetricsRaw, error) {
			t.Fatal("no debe consultar métricas cuando la pertenencia falla")
			return MetricsRaw{}, nil
		},
	}
	svc := NewService(repo, newMockCache(), testLogger())
	h := NewAPIHandler(svc)

	r := gin.New()
	r.Use(apperror.ErrorHandler(testLogger()))
	// Middleware de prueba que simula el JWT fijando un customer_id ajeno.
	r.Use(func(c *gin.Context) { c.Set("customer_id", "other-customer") })
	h.RegisterRoutes(r.Group("/api/v1"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/customers/cust-1/metrics", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetMetrics_API_OwnershipMatch(t *testing.T) {
	// JWT con el mismo customer_id que el :id → se permite y devuelve métricas.
	repo := &mockRepo{
		getMetricsRawFn: func(_ context.Context, _ string) (MetricsRaw, error) {
			return seededRaw(), nil
		},
	}
	svc := NewService(repo, newMockCache(), testLogger())
	h := NewAPIHandler(svc)

	r := gin.New()
	r.Use(apperror.ErrorHandler(testLogger()))
	r.Use(func(c *gin.Context) { c.Set("customer_id", "cust-1") })
	h.RegisterRoutes(r.Group("/api/v1"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/customers/cust-1/metrics", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
