package earnburn

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theluisbolivar/fidel-quick/internal/apperror"
)

// stockBurnFn models the ATOMIC stock decrement that the real repository performs
// in SQL (UPDATE ... WHERE stock_total IS NULL OR redeemed_count < stock_total).
// A mutex serializes the check-and-increment exactly like Postgres serializes the
// conditional UPDATE under the row lock: only the request that observes remaining
// stock wins; the rest get ErrRewardOutOfStock and the burn is rolled back.
//
// stockTotal < 0 means "unlimited" (mirrors stock_total IS NULL), so a burn always
// succeeds and the redeemed counter still tracks how many ran.
func stockBurnFn(stockTotal int, redeemed *int64) func(context.Context, *Transaction, *Redemption) error {
	var mu sync.Mutex
	return func(_ context.Context, _ *Transaction, _ *Redemption) error {
		mu.Lock()
		defer mu.Unlock()
		if stockTotal >= 0 && int(atomic.LoadInt64(redeemed)) >= stockTotal {
			return ErrRewardOutOfStock // agotado: 0 filas afectadas
		}
		atomic.AddInt64(redeemed, 1)
		return nil
	}
}

func newStockRepo(stockTotal int, redeemed *int64) *mockRepo {
	return &mockRepo{
		getRewardFn: func(_ context.Context, id string) (*Reward, error) {
			return &Reward{ID: id, CustomerID: "cust-1", PointsCost: 50, Name: "Cafe"}, nil
		},
		getBalanceFn: func(_ context.Context, _, _ string) (int, error) {
			return 100000, nil // saldo de sobra; el foco es el stock
		},
		burnPointsTxFn: stockBurnFn(stockTotal, redeemed),
	}
}

// TestRequestRedemption_StockDecrements: un canje normal consume una unidad.
func TestRequestRedemption_StockDecrements(t *testing.T) {
	var redeemed int64
	svc := newTestService(newStockRepo(5, &redeemed), newMockCache())

	_, _, err := svc.RequestRedemption(context.Background(), RedemptionReq{
		ClientID: "client-1", CustomerSisfiID: "cs-1", RewardID: "reward-1",
	})

	require.NoError(t, err)
	assert.Equal(t, int64(1), atomic.LoadInt64(&redeemed), "un canje debe consumir exactamente una unidad")
}

// TestRequestRedemption_OutOfStock: un premio agotado rechaza con error tipado y
// NO canjea (el burn se revirtió, el contador no avanza).
func TestRequestRedemption_OutOfStock(t *testing.T) {
	redeemed := int64(3)
	svc := newTestService(newStockRepo(3, &redeemed), newMockCache()) // 3/3 ya canjeados

	_, _, err := svc.RequestRedemption(context.Background(), RedemptionReq{
		ClientID: "client-1", CustomerSisfiID: "cs-1", RewardID: "reward-1",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRewardOutOfStock, "debe envolver ErrRewardOutOfStock")

	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr), "debe ser un apperror tipado")
	assert.Equal(t, 409, appErr.HTTPStatus, "premio agotado => 409 Conflict")
	assert.Contains(t, appErr.Message, "agotado")
	assert.Equal(t, int64(3), atomic.LoadInt64(&redeemed), "un premio agotado no incrementa el contador")
}

// TestRequestRedemption_UnlimitedStock: stock_total NULL (modelado como -1) =
// ilimitado, comportamiento por defecto sin cambio: siempre canjea.
func TestRequestRedemption_UnlimitedStock(t *testing.T) {
	var redeemed int64
	svc := newTestService(newStockRepo(-1, &redeemed), newMockCache())

	const n = 50
	for i := 0; i < n; i++ {
		_, _, err := svc.RequestRedemption(context.Background(), RedemptionReq{
			ClientID: "client-1", CustomerSisfiID: "cs-1", RewardID: "reward-1",
		})
		require.NoError(t, err, "stock ilimitado nunca se agota")
	}
	assert.Equal(t, int64(n), atomic.LoadInt64(&redeemed))
}

// TestRequestRedemption_ConcurrentLastUnit: dos (N) clientes pelean por la ÚLTIMA
// unidad de forma concurrente; solo uno gana, el resto recibe "premio agotado".
// Nunca se sobre-canjea. Esto ejercita la garantía de concurrencia que en
// producción provee el UPDATE condicional atómico de Postgres.
func TestRequestRedemption_ConcurrentLastUnit(t *testing.T) {
	var redeemed int64
	stockTotal := 1 // solo queda 1 unidad
	repo := newStockRepo(stockTotal, &redeemed)
	svc := newTestService(repo, newMockCache())

	const contenders = 8
	var (
		wg          sync.WaitGroup
		successes   int64
		outOfStocks int64
	)
	start := make(chan struct{})
	for i := 0; i < contenders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // liberar todas a la vez para maximizar la contención
			_, _, err := svc.RequestRedemption(context.Background(), RedemptionReq{
				ClientID: "client-1", CustomerSisfiID: "cs-1", RewardID: "reward-1",
			})
			if err == nil {
				atomic.AddInt64(&successes, 1)
			} else if errors.Is(err, ErrRewardOutOfStock) {
				atomic.AddInt64(&outOfStocks, 1)
			} else {
				t.Errorf("error inesperado: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	assert.Equal(t, int64(1), atomic.LoadInt64(&successes), "exactamente un ganador por la última unidad")
	assert.Equal(t, int64(contenders-1), atomic.LoadInt64(&outOfStocks), "el resto recibe premio agotado")
	assert.Equal(t, int64(stockTotal), atomic.LoadInt64(&redeemed), "nunca se sobre-canjea el stock")
}
