package cashback

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theluisbolivar/fidel-quick/internal/modules/earnburn"
)

// --- Mock Repository ---

type mockRepo struct {
	getProgramFn              func(ctx context.Context, customerID string) (*CashbackProgram, error)
	getBalanceFn              func(ctx context.Context, clientID, programID string) (float64, error)
	addCashbackTxFn           func(ctx context.Context, t *CashbackTransaction) (float64, error)
	getTransactionFn          func(ctx context.Context, id string) (*CashbackTransaction, error)
	adjustCashbackTxFn        func(ctx context.Context, t *CashbackTransaction) (float64, error)
	listTransactionsFn        func(ctx context.Context, clientID, programID string, limit int) ([]CashbackTransaction, error)
	getRewardFn               func(ctx context.Context, id string) (*CashbackReward, error)
	listRewardsFn             func(ctx context.Context, customerID, programID string, maxCost float64) ([]CashbackReward, error)
	burnCashbackTxFn          func(ctx context.Context, t *CashbackTransaction, rd *CashbackRedemption) error
	getRedemptionByCodeFn     func(ctx context.Context, code string) (*CashbackRedemption, error)
	confirmRedemptionFn       func(ctx context.Context, id, collaboratorID string) error
	getClientNameFn           func(ctx context.Context, clientID string) (string, error)
	createFeedbackFn          func(ctx context.Context, clientID, customerID, message string) error
	listProgramsFn            func(ctx context.Context, customerID string) ([]CashbackProgram, error)
	createProgramFn           func(ctx context.Context, p *CashbackProgram) error
	updateProgramFn           func(ctx context.Context, p *CashbackProgram) error
	listAllRewardsFn          func(ctx context.Context, programID string) ([]CashbackReward, error)
	createRewardAdminFn       func(ctx context.Context, programID string, r *CashbackReward) error
	updateRewardAdminFn       func(ctx context.Context, r *CashbackReward) error
}

func (m *mockRepo) GetProgram(ctx context.Context, customerID string) (*CashbackProgram, error) {
	if m.getProgramFn != nil {
		return m.getProgramFn(ctx, customerID)
	}
	return &CashbackProgram{ID: "prog-1", CustomerID: customerID, CashbackRate: 0.05, Active: true}, nil
}
func (m *mockRepo) GetBalance(ctx context.Context, clientID, programID string) (float64, error) {
	if m.getBalanceFn != nil {
		return m.getBalanceFn(ctx, clientID, programID)
	}
	return 0, nil
}
func (m *mockRepo) UpsertBalance(ctx context.Context, clientID, programID string, delta float64) (float64, error) {
	return delta, nil
}
func (m *mockRepo) CreateTransaction(ctx context.Context, tx *CashbackTransaction) error { return nil }
func (m *mockRepo) GetTransaction(ctx context.Context, id string) (*CashbackTransaction, error) {
	if m.getTransactionFn != nil {
		return m.getTransactionFn(ctx, id)
	}
	return nil, fmt.Errorf("not found")
}
func (m *mockRepo) ListTransactions(ctx context.Context, clientID, programID string, limit int) ([]CashbackTransaction, error) {
	if m.listTransactionsFn != nil {
		return m.listTransactionsFn(ctx, clientID, programID, limit)
	}
	return nil, nil
}
func (m *mockRepo) ListCorrectableTransactions(ctx context.Context, clientID string) ([]CashbackTransaction, error) {
	return nil, nil
}
func (m *mockRepo) GetClientName(ctx context.Context, clientID string) (string, error) {
	if m.getClientNameFn != nil {
		return m.getClientNameFn(ctx, clientID)
	}
	return "Test Client", nil
}
func (m *mockRepo) ListRewards(ctx context.Context, customerID, programID string, maxCost float64) ([]CashbackReward, error) {
	if m.listRewardsFn != nil {
		return m.listRewardsFn(ctx, customerID, programID, maxCost)
	}
	return nil, nil
}
func (m *mockRepo) GetReward(ctx context.Context, id string) (*CashbackReward, error) {
	if m.getRewardFn != nil {
		return m.getRewardFn(ctx, id)
	}
	return &CashbackReward{ID: id, CustomerID: "cust-1", Cost: 10.0, Name: "Beneficio"}, nil
}
func (m *mockRepo) CreateReward(ctx context.Context, r *CashbackReward) error         { return nil }
func (m *mockRepo) UpdateReward(ctx context.Context, r *CashbackReward) error         { return nil }
func (m *mockRepo) CreateRedemption(ctx context.Context, r *CashbackRedemption) error { return nil }
func (m *mockRepo) GetRedemptionByCode(ctx context.Context, code string) (*CashbackRedemption, error) {
	if m.getRedemptionByCodeFn != nil {
		return m.getRedemptionByCodeFn(ctx, code)
	}
	return nil, fmt.Errorf("not found")
}
func (m *mockRepo) ConfirmRedemption(ctx context.Context, id, collaboratorID string) error {
	if m.confirmRedemptionFn != nil {
		return m.confirmRedemptionFn(ctx, id, collaboratorID)
	}
	return nil
}
func (m *mockRepo) ExpirePendingRedemptions(ctx context.Context) (int, error) { return 0, nil }
func (m *mockRepo) CreateFeedback(ctx context.Context, clientID, customerID, message string) error {
	if m.createFeedbackFn != nil {
		return m.createFeedbackFn(ctx, clientID, customerID, message)
	}
	return nil
}
func (m *mockRepo) ListPrograms(ctx context.Context, customerID string) ([]CashbackProgram, error) {
	if m.listProgramsFn != nil {
		return m.listProgramsFn(ctx, customerID)
	}
	return nil, nil
}
func (m *mockRepo) CreateProgram(ctx context.Context, p *CashbackProgram) error {
	if m.createProgramFn != nil {
		return m.createProgramFn(ctx, p)
	}
	p.ID = "new-prog"
	return nil
}
func (m *mockRepo) UpdateProgram(ctx context.Context, p *CashbackProgram) error {
	if m.updateProgramFn != nil {
		return m.updateProgramFn(ctx, p)
	}
	return nil
}
func (m *mockRepo) ListAllRewards(ctx context.Context, programID string) ([]CashbackReward, error) {
	if m.listAllRewardsFn != nil {
		return m.listAllRewardsFn(ctx, programID)
	}
	return nil, nil
}
func (m *mockRepo) CreateRewardAdmin(ctx context.Context, programID string, r *CashbackReward) error {
	if m.createRewardAdminFn != nil {
		return m.createRewardAdminFn(ctx, programID, r)
	}
	r.ID = "new-reward"
	return nil
}
func (m *mockRepo) UpdateRewardAdmin(ctx context.Context, r *CashbackReward) error {
	if m.updateRewardAdminFn != nil {
		return m.updateRewardAdminFn(ctx, r)
	}
	return nil
}
func (m *mockRepo) AddCashbackTx(ctx context.Context, t *CashbackTransaction) (float64, error) {
	if m.addCashbackTxFn != nil {
		return m.addCashbackTxFn(ctx, t)
	}
	return t.Amount, nil
}
func (m *mockRepo) BurnCashbackTx(ctx context.Context, t *CashbackTransaction, rd *CashbackRedemption) error {
	if m.burnCashbackTxFn != nil {
		return m.burnCashbackTxFn(ctx, t, rd)
	}
	return nil
}
func (m *mockRepo) AdjustCashbackTx(ctx context.Context, t *CashbackTransaction) (float64, error) {
	if m.adjustCashbackTxFn != nil {
		return m.adjustCashbackTxFn(ctx, t)
	}
	return 100.0, nil
}
func (m *mockRepo) EnsureBalance(ctx context.Context, clientID, programID string) error { return nil }
func (m *mockRepo) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error        { return nil }

// --- Mock Cache ---

type mockCache struct {
	otps map[string]*OTPData
}

func newMockCache() *mockCache {
	return &mockCache{otps: make(map[string]*OTPData)}
}

func (c *mockCache) SetOTP(ctx context.Context, code string, data *OTPData) error {
	c.otps[code] = data
	return nil
}
func (c *mockCache) GetOTP(ctx context.Context, code string) (*OTPData, error) {
	if data, ok := c.otps[code]; ok {
		return data, nil
	}
	return nil, nil
}
func (c *mockCache) ConsumeOTP(ctx context.Context, code string) (*OTPData, error) {
	data, ok := c.otps[code]
	if !ok {
		return nil, nil
	}
	delete(c.otps, code)
	return data, nil
}
func (c *mockCache) DeleteOTP(ctx context.Context, code string) error {
	delete(c.otps, code)
	return nil
}

// --- Mock Identity Cache (shared earnburn.Cache) ---

type mockIdentityCache struct {
	otps             map[string]*earnburn.OTPData
	activeIdentities map[string]string
}

func newMockIdentityCache() *mockIdentityCache {
	return &mockIdentityCache{
		otps:             make(map[string]*earnburn.OTPData),
		activeIdentities: make(map[string]string),
	}
}

func (c *mockIdentityCache) SetOTP(ctx context.Context, code string, data *earnburn.OTPData) error {
	c.otps[code] = data
	return nil
}
func (c *mockIdentityCache) GetOTP(ctx context.Context, code string) (*earnburn.OTPData, error) {
	if data, ok := c.otps[code]; ok {
		return data, nil
	}
	return nil, nil
}
func (c *mockIdentityCache) ConsumeOTP(ctx context.Context, code string) (*earnburn.OTPData, error) {
	data, ok := c.otps[code]
	if !ok {
		return nil, nil
	}
	delete(c.otps, code)
	return data, nil
}
func (c *mockIdentityCache) DeleteOTP(ctx context.Context, code string) error {
	delete(c.otps, code)
	return nil
}
func (c *mockIdentityCache) SetActiveIdentity(ctx context.Context, clientID, code string) error {
	c.activeIdentities[clientID] = code
	return nil
}
func (c *mockIdentityCache) GetActiveIdentity(ctx context.Context, clientID string) (string, error) {
	return c.activeIdentities[clientID], nil
}
func (c *mockIdentityCache) DeleteActiveIdentity(ctx context.Context, clientID string) error {
	delete(c.activeIdentities, clientID)
	return nil
}

// --- Test helpers ---

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestService(repo *mockRepo, cache *mockCache, idCache *mockIdentityCache) *Service {
	return NewService(repo, cache, idCache, testLogger())
}

// --- Tests ---

func TestAddCashback_Success(t *testing.T) {
	repo := &mockRepo{
		getProgramFn: func(_ context.Context, _ string) (*CashbackProgram, error) {
			return &CashbackProgram{ID: "prog-1", CashbackRate: 0.05}, nil
		},
		addCashbackTxFn: func(_ context.Context, tx *CashbackTransaction) (float64, error) {
			assert.Equal(t, "earn", tx.Type)
			assert.Equal(t, 5.0, tx.Amount) // 100 * 0.05 = 5.0
			return 5.0, nil
		},
	}
	svc := newTestService(repo, newMockCache(), newMockIdentityCache())

	tx, err := svc.AddCashback(context.Background(), AddCashbackReq{
		ClientID:  "client-1",
		ProgramID: "prog-1",
		Amount:    100.0,
	})

	require.NoError(t, err)
	assert.Equal(t, 5.0, tx.Amount)
	assert.Equal(t, 5.0, tx.BalanceAfter)
	assert.Equal(t, "earn", tx.Type)
	assert.NotNil(t, tx.CorrectableUntil)
}

func TestAddCashback_InsufficientAmount(t *testing.T) {
	repo := &mockRepo{
		getProgramFn: func(_ context.Context, _ string) (*CashbackProgram, error) {
			return &CashbackProgram{ID: "prog-1", CashbackRate: 0.05}, nil
		},
	}
	svc := newTestService(repo, newMockCache(), newMockIdentityCache())

	_, err := svc.AddCashback(context.Background(), AddCashbackReq{
		ClientID: "client-1",
		Amount:   0.0,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "monto insuficiente")
}

func TestCheckBalance(t *testing.T) {
	repo := &mockRepo{
		getBalanceFn: func(_ context.Context, _, _ string) (float64, error) {
			return 42.50, nil
		},
	}
	svc := newTestService(repo, newMockCache(), newMockIdentityCache())

	balance, err := svc.CheckBalance(context.Background(), "client-1", "prog-1")

	require.NoError(t, err)
	assert.Equal(t, 42.50, balance)
}

func TestUpdateCashback_Success(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	repo := &mockRepo{
		getTransactionFn: func(_ context.Context, id string) (*CashbackTransaction, error) {
			return &CashbackTransaction{
				ID: id, ClientID: "client-1", ProgramID: "prog-1",
				Amount: 5.0, CorrectableUntil: &future,
			}, nil
		},
		getProgramFn: func(_ context.Context, _ string) (*CashbackProgram, error) {
			return &CashbackProgram{CashbackRate: 0.05}, nil
		},
		adjustCashbackTxFn: func(_ context.Context, tx *CashbackTransaction) (float64, error) {
			assert.Equal(t, "adjustment", tx.Type)
			// new purchase 200 * 0.05 = 10, original was 5, delta = 5
			assert.Equal(t, 5.0, tx.Amount)
			return 10.0, nil
		},
	}
	svc := newTestService(repo, newMockCache(), newMockIdentityCache())

	tx, err := svc.UpdateCashback(context.Background(), UpdateCashbackReq{
		TransactionID:     "tx-1",
		NewPurchaseAmount: 200.0,
	})

	require.NoError(t, err)
	assert.Equal(t, 5.0, tx.Amount)
	assert.Equal(t, 10.0, tx.BalanceAfter)
}

func TestUpdateCashback_Expired(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	repo := &mockRepo{
		getTransactionFn: func(_ context.Context, _ string) (*CashbackTransaction, error) {
			return &CashbackTransaction{
				ID: "tx-1", Amount: 5.0, CorrectableUntil: &past,
			}, nil
		},
	}
	svc := newTestService(repo, newMockCache(), newMockIdentityCache())

	_, err := svc.UpdateCashback(context.Background(), UpdateCashbackReq{
		TransactionID:     "tx-1",
		NewPurchaseAmount: 200.0,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ventana de correccion expirada")
}

func TestRequestRedemption_Success(t *testing.T) {
	repo := &mockRepo{
		getRewardFn: func(_ context.Context, id string) (*CashbackReward, error) {
			return &CashbackReward{ID: id, CustomerID: "cust-1", Cost: 10.0, Name: "Descuento"}, nil
		},
		getBalanceFn: func(_ context.Context, _, _ string) (float64, error) {
			return 50.0, nil
		},
		burnCashbackTxFn: func(_ context.Context, tx *CashbackTransaction, rd *CashbackRedemption) error {
			assert.Equal(t, "burn", tx.Type)
			assert.Equal(t, -10.0, tx.Amount)
			assert.Equal(t, "pending", rd.Status)
			return nil
		},
	}
	cache := newMockCache()
	svc := newTestService(repo, cache, newMockIdentityCache())

	rd, code, err := svc.RequestRedemption(context.Background(), CashbackRedemptionReq{
		ClientID:  "client-1",
		ProgramID: "prog-1",
		RewardID:  "reward-1",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, code)
	assert.Equal(t, 6, len(code))
	assert.Equal(t, "pending", rd.Status)
	assert.Equal(t, 10.0, rd.AmountSpent)

	otpData := cache.otps[code]
	assert.Equal(t, "cb_redemption", otpData.Type)
}

func TestRequestRedemption_InsufficientBalance(t *testing.T) {
	repo := &mockRepo{
		getRewardFn: func(_ context.Context, _ string) (*CashbackReward, error) {
			return &CashbackReward{Cost: 100.0}, nil
		},
		getBalanceFn: func(_ context.Context, _, _ string) (float64, error) {
			return 50.0, nil
		},
	}
	svc := newTestService(repo, newMockCache(), newMockIdentityCache())

	_, _, err := svc.RequestRedemption(context.Background(), CashbackRedemptionReq{
		ClientID: "client-1",
		RewardID: "reward-1",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "saldo insuficiente")
}

func TestConfirmRedemption_Success(t *testing.T) {
	repo := &mockRepo{
		getRedemptionByCodeFn: func(_ context.Context, code string) (*CashbackRedemption, error) {
			return &CashbackRedemption{
				ID: "rd-1", Code: code, Status: "pending",
				ExpiresAt: time.Now().Add(1 * time.Hour),
			}, nil
		},
	}
	cache := newMockCache()
	cache.otps["ABC123"] = &OTPData{Type: "cb_redemption"}
	svc := newTestService(repo, cache, newMockIdentityCache())

	rd, err := svc.ConfirmRedemption(context.Background(), "ABC123", "collab-1")

	require.NoError(t, err)
	assert.Equal(t, "confirmed", rd.Status)
	assert.Equal(t, "collab-1", rd.ConfirmedBy)
}

func TestConfirmRedemption_AlreadyConfirmed(t *testing.T) {
	repo := &mockRepo{
		getRedemptionByCodeFn: func(_ context.Context, _ string) (*CashbackRedemption, error) {
			return &CashbackRedemption{Status: "confirmed"}, nil
		},
	}
	svc := newTestService(repo, newMockCache(), newMockIdentityCache())

	_, err := svc.ConfirmRedemption(context.Background(), "ABC123", "collab-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "canje ya fue confirmed")
}

func TestConfirmRedemption_Expired(t *testing.T) {
	repo := &mockRepo{
		getRedemptionByCodeFn: func(_ context.Context, _ string) (*CashbackRedemption, error) {
			return &CashbackRedemption{
				Status:    "pending",
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			}, nil
		},
	}
	svc := newTestService(repo, newMockCache(), newMockIdentityCache())

	_, err := svc.ConfirmRedemption(context.Background(), "ABC123", "collab-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "codigo expirado")
}

func TestRequestLoadCode(t *testing.T) {
	cache := newMockCache()
	svc := newTestService(&mockRepo{}, cache, newMockIdentityCache())

	code, err := svc.RequestLoadCode(context.Background(), "client-1", "cust-1")

	require.NoError(t, err)
	assert.Equal(t, 6, len(code))

	otpData := cache.otps[code]
	assert.Equal(t, "cb_load_points", otpData.Type)
}

func TestValidateLoadCode_Success(t *testing.T) {
	cache := newMockCache()
	cache.otps["ABC123"] = &OTPData{ClientID: "client-1", Type: "cb_load_points"}
	svc := newTestService(&mockRepo{}, cache, newMockIdentityCache())

	data, err := svc.ValidateLoadCode(context.Background(), "ABC123")

	require.NoError(t, err)
	assert.Equal(t, "client-1", data.ClientID)

	_, ok := cache.otps["ABC123"]
	assert.False(t, ok) // consumed
}

func TestValidateLoadCode_WrongType(t *testing.T) {
	cache := newMockCache()
	cache.otps["ABC123"] = &OTPData{Type: "cb_redemption"}
	svc := newTestService(&mockRepo{}, cache, newMockIdentityCache())

	_, err := svc.ValidateLoadCode(context.Background(), "ABC123")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "tipo incorrecto")
}

func TestRequestIdentityOTP(t *testing.T) {
	idCache := newMockIdentityCache()
	svc := newTestService(&mockRepo{}, newMockCache(), idCache)

	code, err := svc.RequestIdentityOTP(context.Background(), "client-1", "cust-1")

	require.NoError(t, err)
	assert.Equal(t, 6, len(code))

	otpData := idCache.otps[code]
	assert.Equal(t, "identity", otpData.Type)
	assert.Equal(t, code, idCache.activeIdentities["client-1"])
}

func TestValidateIdentityOTP_Success(t *testing.T) {
	idCache := newMockIdentityCache()
	idCache.otps["ABC123"] = &earnburn.OTPData{ClientID: "client-1", Type: "identity"}
	svc := newTestService(&mockRepo{}, newMockCache(), idCache)

	data, err := svc.ValidateIdentityOTP(context.Background(), "ABC123")

	require.NoError(t, err)
	assert.Equal(t, "client-1", data.ClientID)

	// Should NOT be consumed
	_, ok := idCache.otps["ABC123"]
	assert.True(t, ok)
}

func TestSubmitFeedback(t *testing.T) {
	var called bool
	repo := &mockRepo{
		createFeedbackFn: func(_ context.Context, clientID, customerID, message string) error {
			called = true
			assert.Equal(t, "Excelente", message)
			return nil
		},
	}
	svc := newTestService(repo, newMockCache(), newMockIdentityCache())

	err := svc.SubmitFeedback(context.Background(), "client-1", "cust-1", "Excelente")

	require.NoError(t, err)
	assert.True(t, called)
}

func TestGetProgram_NotFound(t *testing.T) {
	repo := &mockRepo{
		getProgramFn: func(_ context.Context, _ string) (*CashbackProgram, error) {
			return nil, sql.ErrNoRows
		},
	}
	svc := newTestService(repo, newMockCache(), newMockIdentityCache())

	_, err := svc.GetProgram(context.Background(), "cust-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "programa cashback no encontrado")
}

func TestListPrograms(t *testing.T) {
	repo := &mockRepo{
		listProgramsFn: func(_ context.Context, _ string) ([]CashbackProgram, error) {
			return []CashbackProgram{{ID: "p-1"}, {ID: "p-2"}}, nil
		},
	}
	svc := newTestService(repo, newMockCache(), newMockIdentityCache())

	programs, err := svc.ListPrograms(context.Background(), "cust-1")

	require.NoError(t, err)
	assert.Len(t, programs, 2)
}

func TestCreateProgram(t *testing.T) {
	repo := &mockRepo{
		createProgramFn: func(_ context.Context, p *CashbackProgram) error {
			p.ID = "new-prog"
			return nil
		},
	}
	svc := newTestService(repo, newMockCache(), newMockIdentityCache())

	p := &CashbackProgram{CustomerID: "cust-1", Name: "Test CB"}
	err := svc.CreateProgram(context.Background(), p)

	require.NoError(t, err)
	assert.Equal(t, "new-prog", p.ID)
}
