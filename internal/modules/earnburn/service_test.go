package earnburn

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
)

// --- Mock Repository ---

type mockRepo struct {
	getProgramFn                   func(ctx context.Context, customerID string) (*EarnBurnProgram, error)
	getBalanceFn                   func(ctx context.Context, clientID, customerSisfiID string) (int, error)
	addPointsTxFn                  func(ctx context.Context, t *Transaction) (int, error)
	getTransactionFn               func(ctx context.Context, id string) (*Transaction, error)
	adjustPointsTxFn               func(ctx context.Context, t *Transaction) (int, error)
	listTransactionsFn             func(ctx context.Context, clientID, customerSisfiID string, limit int) ([]Transaction, error)
	listCorrectableTransactionsFn  func(ctx context.Context, clientID string) ([]Transaction, error)
	getRewardFn                    func(ctx context.Context, id string) (*Reward, error)
	listRewardsFn                  func(ctx context.Context, customerID, customerSisfiID string, maxPoints int) ([]Reward, error)
	burnPointsTxFn                 func(ctx context.Context, t *Transaction, rd *Redemption) error
	getRedemptionByCodeFn          func(ctx context.Context, code string) (*Redemption, error)
	confirmRedemptionFn            func(ctx context.Context, id, collaboratorID string) error
	getClientNameFn                func(ctx context.Context, clientID string) (string, error)
	createFeedbackFn               func(ctx context.Context, clientID, customerID, message string) error
	listProgramsFn                 func(ctx context.Context, customerID string) ([]EarnBurnProgram, error)
	getCustomerFn                  func(ctx context.Context, id string) (*Customer, error)
	createCustomerFn               func(ctx context.Context, c *Customer) error
	updateCustomerFn               func(ctx context.Context, c *Customer) error
	createCollaboratorFn           func(ctx context.Context, c *Collaborator) error
	listCollaboratorsFn            func(ctx context.Context, customerID string) ([]Collaborator, error)
	listAllRewardsFn               func(ctx context.Context, customerSisfiID string) ([]Reward, error)
	createRewardAdminFn            func(ctx context.Context, customerSisfiID string, r *Reward) error
	updateRewardAdminFn            func(ctx context.Context, r *Reward) error
	listFeedbackFn                 func(ctx context.Context, customerID string) ([]FeedbackEntry, error)
	listClientsFn                  func(ctx context.Context, customerID string) ([]Client, error)
	updateProgramFn                func(ctx context.Context, p *EarnBurnProgram, setActive *bool) error
	expirePointsFn                 func(ctx context.Context, clientID, customerSisfiID string, expiryDays int) (int, error)
}

func (m *mockRepo) GetProgram(ctx context.Context, customerID string) (*EarnBurnProgram, error) {
	if m.getProgramFn != nil {
		return m.getProgramFn(ctx, customerID)
	}
	return &EarnBurnProgram{CustomerSisfiID: "cs-1", CustomerID: customerID, PointsRatio: 10, Active: true}, nil
}
func (m *mockRepo) GetProgramByID(ctx context.Context, customerSisfiID string) (*EarnBurnProgram, error) {
	if m.getProgramFn != nil {
		return m.getProgramFn(ctx, customerSisfiID)
	}
	return &EarnBurnProgram{CustomerSisfiID: customerSisfiID, CustomerID: "cust-1", PointsRatio: 10, Active: true}, nil
}

func (m *mockRepo) GetBalance(ctx context.Context, clientID, customerSisfiID string) (int, error) {
	if m.getBalanceFn != nil {
		return m.getBalanceFn(ctx, clientID, customerSisfiID)
	}
	return 0, nil
}

func (m *mockRepo) UpsertBalance(ctx context.Context, clientID, customerSisfiID string, delta int) (int, error) {
	return delta, nil
}

func (m *mockRepo) CreateTransaction(ctx context.Context, tx *Transaction) error { return nil }

func (m *mockRepo) GetTransaction(ctx context.Context, id string) (*Transaction, error) {
	if m.getTransactionFn != nil {
		return m.getTransactionFn(ctx, id)
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockRepo) ListTransactions(ctx context.Context, clientID, customerSisfiID string, limit int) ([]Transaction, error) {
	if m.listTransactionsFn != nil {
		return m.listTransactionsFn(ctx, clientID, customerSisfiID, limit)
	}
	return nil, nil
}

func (m *mockRepo) ListCorrectableTransactions(ctx context.Context, clientID string) ([]Transaction, error) {
	if m.listCorrectableTransactionsFn != nil {
		return m.listCorrectableTransactionsFn(ctx, clientID)
	}
	return nil, nil
}

func (m *mockRepo) GetClientName(ctx context.Context, clientID string) (string, error) {
	if m.getClientNameFn != nil {
		return m.getClientNameFn(ctx, clientID)
	}
	return "Test Client", nil
}

func (m *mockRepo) ListRewards(ctx context.Context, customerID, customerSisfiID string, maxPoints int) ([]Reward, error) {
	if m.listRewardsFn != nil {
		return m.listRewardsFn(ctx, customerID, customerSisfiID, maxPoints)
	}
	return nil, nil
}

func (m *mockRepo) GetReward(ctx context.Context, id string) (*Reward, error) {
	if m.getRewardFn != nil {
		return m.getRewardFn(ctx, id)
	}
	return &Reward{ID: id, CustomerID: "cust-1", PointsCost: 100, Name: "Premio"}, nil
}

func (m *mockRepo) CreateReward(ctx context.Context, r *Reward) error { return nil }
func (m *mockRepo) UpdateReward(ctx context.Context, r *Reward) error { return nil }

func (m *mockRepo) CreateRedemption(ctx context.Context, r *Redemption) error { return nil }

func (m *mockRepo) GetRedemptionByCode(ctx context.Context, code string) (*Redemption, error) {
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

func (m *mockRepo) ListPrograms(ctx context.Context, customerID string) ([]EarnBurnProgram, error) {
	if m.listProgramsFn != nil {
		return m.listProgramsFn(ctx, customerID)
	}
	return []EarnBurnProgram{{CustomerSisfiID: "cs-1", CustomerID: customerID}}, nil
}

func (m *mockRepo) CreateProgram(ctx context.Context, p *EarnBurnProgram) error { return nil }

func (m *mockRepo) GetCustomer(ctx context.Context, id string) (*Customer, error) {
	if m.getCustomerFn != nil {
		return m.getCustomerFn(ctx, id)
	}
	return &Customer{ID: id, Name: "Test Business"}, nil
}

func (m *mockRepo) CreateCustomer(ctx context.Context, c *Customer) error {
	if m.createCustomerFn != nil {
		return m.createCustomerFn(ctx, c)
	}
	c.ID = "new-cust"
	return nil
}

func (m *mockRepo) UpdateCustomer(ctx context.Context, c *Customer) error {
	if m.updateCustomerFn != nil {
		return m.updateCustomerFn(ctx, c)
	}
	return nil
}

func (m *mockRepo) CreateCollaborator(ctx context.Context, c *Collaborator) error {
	if m.createCollaboratorFn != nil {
		return m.createCollaboratorFn(ctx, c)
	}
	c.ID = "new-collab"
	return nil
}

func (m *mockRepo) ListCollaborators(ctx context.Context, customerID string) ([]Collaborator, error) {
	if m.listCollaboratorsFn != nil {
		return m.listCollaboratorsFn(ctx, customerID)
	}
	return nil, nil
}

func (m *mockRepo) ListAllRewards(ctx context.Context, customerSisfiID string) ([]Reward, error) {
	if m.listAllRewardsFn != nil {
		return m.listAllRewardsFn(ctx, customerSisfiID)
	}
	return nil, nil
}

func (m *mockRepo) CreateRewardAdmin(ctx context.Context, customerSisfiID string, r *Reward) error {
	if m.createRewardAdminFn != nil {
		return m.createRewardAdminFn(ctx, customerSisfiID, r)
	}
	r.ID = "new-reward"
	return nil
}

func (m *mockRepo) UpdateRewardAdmin(ctx context.Context, r *Reward) error {
	if m.updateRewardAdminFn != nil {
		return m.updateRewardAdminFn(ctx, r)
	}
	return nil
}

func (m *mockRepo) ListFeedback(ctx context.Context, customerID string) ([]FeedbackEntry, error) {
	if m.listFeedbackFn != nil {
		return m.listFeedbackFn(ctx, customerID)
	}
	return nil, nil
}

func (m *mockRepo) ListClients(ctx context.Context, customerID string) ([]Client, error) {
	if m.listClientsFn != nil {
		return m.listClientsFn(ctx, customerID)
	}
	return nil, nil
}

func (m *mockRepo) RegisterClient(ctx context.Context, customerID, phone string) error { return nil }

func (m *mockRepo) AddPointsTx(ctx context.Context, t *Transaction) (int, error) {
	if m.addPointsTxFn != nil {
		return m.addPointsTxFn(ctx, t)
	}
	return t.Amount, nil
}

func (m *mockRepo) BurnPointsTx(ctx context.Context, t *Transaction, rd *Redemption) error {
	if m.burnPointsTxFn != nil {
		return m.burnPointsTxFn(ctx, t, rd)
	}
	return nil
}

func (m *mockRepo) AdjustPointsTx(ctx context.Context, t *Transaction) (int, error) {
	if m.adjustPointsTxFn != nil {
		return m.adjustPointsTxFn(ctx, t)
	}
	return 100, nil
}

func (m *mockRepo) EnsureBalance(ctx context.Context, clientID, customerSisfiID string) error {
	return nil
}

func (m *mockRepo) UpdateProgram(ctx context.Context, p *EarnBurnProgram, setActive *bool) error {
	if m.updateProgramFn != nil {
		return m.updateProgramFn(ctx, p, setActive)
	}
	return nil
}

func (m *mockRepo) ExpirePoints(ctx context.Context, clientID, customerSisfiID string, expiryDays int) (int, error) {
	if m.expirePointsFn != nil {
		return m.expirePointsFn(ctx, clientID, customerSisfiID, expiryDays)
	}
	return 0, nil
}
func (m *mockRepo) GetClientPhone(ctx context.Context, clientID string) (string, error) {
	return "", nil
}

// --- Mock Cache ---

type mockCache struct {
	otps             map[string]*OTPData
	activeIdentities map[string]string
	setOTPFn         func(ctx context.Context, code string, data *OTPData) error
	consumeOTPFn     func(ctx context.Context, code string) (*OTPData, error)
}

func newMockCache() *mockCache {
	return &mockCache{
		otps:             make(map[string]*OTPData),
		activeIdentities: make(map[string]string),
	}
}

func (c *mockCache) SetOTP(ctx context.Context, code string, data *OTPData) error {
	if c.setOTPFn != nil {
		return c.setOTPFn(ctx, code, data)
	}
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
	if c.consumeOTPFn != nil {
		return c.consumeOTPFn(ctx, code)
	}
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

func (c *mockCache) SetActiveIdentity(ctx context.Context, clientID, code string) error {
	c.activeIdentities[clientID] = code
	return nil
}

func (c *mockCache) GetActiveIdentity(ctx context.Context, clientID string) (string, error) {
	return c.activeIdentities[clientID], nil
}

func (c *mockCache) DeleteActiveIdentity(ctx context.Context, clientID string) error {
	delete(c.activeIdentities, clientID)
	return nil
}

// --- Test helpers ---

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestService(repo *mockRepo, cache *mockCache) *Service {
	return NewService(repo, cache, testLogger())
}

// --- Tests ---

func TestAddPoints_Success(t *testing.T) {
	repo := &mockRepo{
		getProgramFn: func(_ context.Context, _ string) (*EarnBurnProgram, error) {
			return &EarnBurnProgram{CustomerSisfiID: "cs-1", PointsRatio: 15}, nil
		},
		addPointsTxFn: func(_ context.Context, tx *Transaction) (int, error) {
			assert.Equal(t, "earn", tx.Type)
			assert.Equal(t, 10, tx.Amount) // 150 / 15 = 10
			return 10, nil
		},
	}
	cache := newMockCache()
	svc := newTestService(repo, cache)

	tx, err := svc.AddPoints(context.Background(), AddPointsReq{
		ClientID:        "client-1",
		CustomerSisfiID: "cs-1",
		Amount:          150, // 150 / 15 ratio = 10 points
	})

	require.NoError(t, err)
	assert.Equal(t, 10, tx.Amount)
	assert.Equal(t, 10, tx.BalanceAfter)
	assert.Equal(t, "earn", tx.Type)
	assert.NotNil(t, tx.CorrectableUntil)
}

func TestAddPoints_InsufficientAmount(t *testing.T) {
	repo := &mockRepo{
		getProgramFn: func(_ context.Context, _ string) (*EarnBurnProgram, error) {
			return &EarnBurnProgram{CustomerSisfiID: "cs-1", PointsRatio: 15}, nil
		},
	}
	svc := newTestService(repo, newMockCache())

	_, err := svc.AddPoints(context.Background(), AddPointsReq{
		ClientID: "client-1",
		Amount:   5, // 5 / 15 = 0 points
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "monto insuficiente")
}

func TestAddPoints_ProgramNotFound(t *testing.T) {
	repo := &mockRepo{
		getProgramFn: func(_ context.Context, _ string) (*EarnBurnProgram, error) {
			return nil, fmt.Errorf("get program: %w", sql.ErrNoRows)
		},
	}
	svc := newTestService(repo, newMockCache())

	_, err := svc.AddPoints(context.Background(), AddPointsReq{
		ClientID: "client-1",
		Amount:   5000,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "get program")
}

func TestCheckBalance(t *testing.T) {
	repo := &mockRepo{
		getBalanceFn: func(_ context.Context, _, _ string) (int, error) {
			return 42, nil
		},
	}
	svc := newTestService(repo, newMockCache())

	balance, err := svc.CheckBalance(context.Background(), "client-1", "cs-1")

	require.NoError(t, err)
	assert.Equal(t, 42, balance)
}

func TestListTransactions(t *testing.T) {
	now := time.Now()
	repo := &mockRepo{
		listTransactionsFn: func(_ context.Context, _, _ string, limit int) ([]Transaction, error) {
			return []Transaction{
				{ID: "tx-1", Type: "earn", Amount: 10, CreatedAt: now},
				{ID: "tx-2", Type: "burn", Amount: -5, CreatedAt: now},
			}, nil
		},
	}
	svc := newTestService(repo, newMockCache())

	txs, err := svc.ListTransactions(context.Background(), "client-1", "cs-1", 10)

	require.NoError(t, err)
	assert.Len(t, txs, 2)
	assert.Equal(t, "earn", txs[0].Type)
	assert.Equal(t, "burn", txs[1].Type)
}

func TestUpdatePoints_Success(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	repo := &mockRepo{
		getTransactionFn: func(_ context.Context, id string) (*Transaction, error) {
			return &Transaction{
				ID: id, ClientID: "client-1", CustomerSisfiID: "cs-1",
				Amount: 10, CorrectableUntil: &future,
			}, nil
		},
		adjustPointsTxFn: func(_ context.Context, tx *Transaction) (int, error) {
			assert.Equal(t, "adjustment", tx.Type)
			assert.Equal(t, 5, tx.Amount) // 15 - 10 = 5 delta
			return 15, nil
		},
	}
	svc := newTestService(repo, newMockCache())

	tx, err := svc.UpdatePoints(context.Background(), UpdatePointsReq{
		TransactionID: "tx-1",
		NewAmount:     15,
	})

	require.NoError(t, err)
	assert.Equal(t, 5, tx.Amount)
	assert.Equal(t, 15, tx.BalanceAfter)
}

func TestUpdatePoints_Expired(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	repo := &mockRepo{
		getTransactionFn: func(_ context.Context, _ string) (*Transaction, error) {
			return &Transaction{
				ID: "tx-1", Amount: 10, CorrectableUntil: &past,
			}, nil
		},
	}
	svc := newTestService(repo, newMockCache())

	_, err := svc.UpdatePoints(context.Background(), UpdatePointsReq{
		TransactionID: "tx-1",
		NewAmount:     15,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ventana de correccion expirada")
}

func TestRequestRedemption_Success(t *testing.T) {
	repo := &mockRepo{
		getRewardFn: func(_ context.Context, id string) (*Reward, error) {
			return &Reward{ID: id, CustomerID: "cust-1", PointsCost: 50, Name: "Cafe"}, nil
		},
		getBalanceFn: func(_ context.Context, _, _ string) (int, error) {
			return 100, nil
		},
		burnPointsTxFn: func(_ context.Context, tx *Transaction, rd *Redemption) error {
			assert.Equal(t, "burn", tx.Type)
			assert.Equal(t, -50, tx.Amount)
			assert.Equal(t, "pending", rd.Status)
			return nil
		},
	}
	cache := newMockCache()
	svc := newTestService(repo, cache)

	rd, code, err := svc.RequestRedemption(context.Background(), RedemptionReq{
		ClientID:        "client-1",
		CustomerSisfiID: "cs-1",
		RewardID:        "reward-1",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, code)
	assert.Equal(t, 6, len(code))
	assert.Equal(t, "pending", rd.Status)
	assert.Equal(t, 50, rd.PointsSpent)

	// OTP should be cached
	otpData, ok := cache.otps[code]
	assert.True(t, ok)
	assert.Equal(t, "redemption", otpData.Type)
}

func TestRequestRedemption_InsufficientBalance(t *testing.T) {
	repo := &mockRepo{
		getRewardFn: func(_ context.Context, _ string) (*Reward, error) {
			return &Reward{ID: "r-1", PointsCost: 100}, nil
		},
		getBalanceFn: func(_ context.Context, _, _ string) (int, error) {
			return 50, nil
		},
	}
	svc := newTestService(repo, newMockCache())

	_, _, err := svc.RequestRedemption(context.Background(), RedemptionReq{
		ClientID:        "client-1",
		CustomerSisfiID: "cs-1",
		RewardID:        "reward-1",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "puntos insuficientes")
}

func TestConfirmRedemption_Success(t *testing.T) {
	repo := &mockRepo{
		getRedemptionByCodeFn: func(_ context.Context, code string) (*Redemption, error) {
			return &Redemption{
				ID: "rd-1", Code: code, Status: "pending",
				ExpiresAt: time.Now().Add(1 * time.Hour),
			}, nil
		},
		confirmRedemptionFn: func(_ context.Context, id, collabID string) error {
			assert.Equal(t, "rd-1", id)
			assert.Equal(t, "collab-1", collabID)
			return nil
		},
	}
	cache := newMockCache()
	cache.otps["ABC123"] = &OTPData{Type: "redemption"}
	svc := newTestService(repo, cache)

	rd, err := svc.ConfirmRedemption(context.Background(), "ABC123", "collab-1")

	require.NoError(t, err)
	assert.Equal(t, "confirmed", rd.Status)
	assert.Equal(t, "collab-1", rd.ConfirmedBy)
}

func TestConfirmRedemption_AlreadyConfirmed(t *testing.T) {
	repo := &mockRepo{
		getRedemptionByCodeFn: func(_ context.Context, _ string) (*Redemption, error) {
			return &Redemption{Status: "confirmed"}, nil
		},
	}
	svc := newTestService(repo, newMockCache())

	_, err := svc.ConfirmRedemption(context.Background(), "ABC123", "collab-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "canje ya fue confirmed")
}

func TestConfirmRedemption_Expired(t *testing.T) {
	repo := &mockRepo{
		getRedemptionByCodeFn: func(_ context.Context, _ string) (*Redemption, error) {
			return &Redemption{
				Status:    "pending",
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			}, nil
		},
	}
	svc := newTestService(repo, newMockCache())

	_, err := svc.ConfirmRedemption(context.Background(), "ABC123", "collab-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "codigo expirado")
}

func TestRequestLoadPointsCode(t *testing.T) {
	cache := newMockCache()
	svc := newTestService(&mockRepo{}, cache)

	code, err := svc.RequestLoadPointsCode(context.Background(), "client-1", "cust-1")

	require.NoError(t, err)
	assert.Equal(t, 6, len(code))

	otpData, ok := cache.otps[code]
	assert.True(t, ok)
	assert.Equal(t, "load_points", otpData.Type)
	assert.Equal(t, "client-1", otpData.ClientID)
}

func TestValidateLoadPointsCode_Success(t *testing.T) {
	cache := newMockCache()
	cache.otps["ABC123"] = &OTPData{ClientID: "client-1", CustomerID: "cust-1", Type: "load_points"}
	svc := newTestService(&mockRepo{}, cache)

	data, err := svc.ValidateLoadPointsCode(context.Background(), "ABC123")

	require.NoError(t, err)
	assert.Equal(t, "client-1", data.ClientID)
	assert.Equal(t, "load_points", data.Type)

	// Should be consumed (deleted)
	_, ok := cache.otps["ABC123"]
	assert.False(t, ok)
}

func TestValidateLoadPointsCode_Invalid(t *testing.T) {
	svc := newTestService(&mockRepo{}, newMockCache())

	_, err := svc.ValidateLoadPointsCode(context.Background(), "INVALID")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "codigo invalido o expirado")
}

func TestValidateLoadPointsCode_WrongType(t *testing.T) {
	cache := newMockCache()
	cache.otps["ABC123"] = &OTPData{Type: "identity"}
	svc := newTestService(&mockRepo{}, cache)

	_, err := svc.ValidateLoadPointsCode(context.Background(), "ABC123")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "tipo incorrecto")
}

func TestRequestIdentityOTP(t *testing.T) {
	cache := newMockCache()
	svc := newTestService(&mockRepo{}, cache)

	code, err := svc.RequestIdentityOTP(context.Background(), "client-1", "cust-1")

	require.NoError(t, err)
	assert.Equal(t, 6, len(code))

	otpData := cache.otps[code]
	assert.Equal(t, "identity", otpData.Type)

	// Active identity should be tracked
	activeCode := cache.activeIdentities["client-1"]
	assert.Equal(t, code, activeCode)
}

func TestRequestIdentityOTP_InvalidatesPrevious(t *testing.T) {
	cache := newMockCache()
	cache.otps["OLD123"] = &OTPData{Type: "identity"}
	cache.activeIdentities["client-1"] = "OLD123"
	svc := newTestService(&mockRepo{}, cache)

	code, err := svc.RequestIdentityOTP(context.Background(), "client-1", "cust-1")

	require.NoError(t, err)

	// Old OTP should be deleted
	_, oldExists := cache.otps["OLD123"]
	assert.False(t, oldExists)

	// New OTP should exist
	_, newExists := cache.otps[code]
	assert.True(t, newExists)
}

func TestValidateIdentityOTP_Success(t *testing.T) {
	cache := newMockCache()
	cache.otps["ABC123"] = &OTPData{ClientID: "client-1", Type: "identity"}
	svc := newTestService(&mockRepo{}, cache)

	data, err := svc.ValidateIdentityOTP(context.Background(), "ABC123")

	require.NoError(t, err)
	assert.Equal(t, "client-1", data.ClientID)

	// Identity OTP should NOT be consumed (multi-use)
	_, stillExists := cache.otps["ABC123"]
	assert.True(t, stillExists)
}

func TestValidateIdentityOTP_Invalid(t *testing.T) {
	svc := newTestService(&mockRepo{}, newMockCache())

	_, err := svc.ValidateIdentityOTP(context.Background(), "INVALID")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "codigo invalido o expirado")
}

func TestSubmitFeedback(t *testing.T) {
	var called bool
	repo := &mockRepo{
		createFeedbackFn: func(_ context.Context, clientID, customerID, message string) error {
			assert.Equal(t, "client-1", clientID)
			assert.Equal(t, "cust-1", customerID)
			assert.Equal(t, "Great service!", message)
			called = true
			return nil
		},
	}
	svc := newTestService(repo, newMockCache())

	err := svc.SubmitFeedback(context.Background(), "client-1", "cust-1", "Great service!")

	require.NoError(t, err)
	assert.True(t, called)
}

func TestGetProgram_NotFound(t *testing.T) {
	repo := &mockRepo{
		getProgramFn: func(_ context.Context, _ string) (*EarnBurnProgram, error) {
			return nil, sql.ErrNoRows
		},
	}
	svc := newTestService(repo, newMockCache())

	_, err := svc.GetProgram(context.Background(), "cust-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "programa no encontrado")
}

func TestListPrograms(t *testing.T) {
	repo := &mockRepo{
		listProgramsFn: func(_ context.Context, customerID string) ([]EarnBurnProgram, error) {
			return []EarnBurnProgram{
				{CustomerSisfiID: "cs-1", CustomerID: customerID},
				{CustomerSisfiID: "cs-2", CustomerID: customerID},
			}, nil
		},
	}
	svc := newTestService(repo, newMockCache())

	programs, err := svc.ListPrograms(context.Background(), "cust-1")

	require.NoError(t, err)
	assert.Len(t, programs, 2)
}

func TestGenerateCode(t *testing.T) {
	code := generateCode(6)
	assert.Equal(t, 6, len(code))

	// Should only contain allowed characters
	allowed := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	for _, c := range code {
		assert.Contains(t, allowed, string(c))
	}

	// Two codes should be different
	code2 := generateCode(6)
	assert.NotEqual(t, code, code2)
}

func TestGenerateUUID(t *testing.T) {
	uuid := generateUUID()
	assert.Equal(t, 36, len(uuid)) // 8-4-4-4-12 + 4 hyphens

	// Two UUIDs should be different
	uuid2 := generateUUID()
	assert.NotEqual(t, uuid, uuid2)
}

// --- FID-36: ticket mínimo (earn_burn) ---

func TestAddPoints_BelowMinTicket_NotCredited(t *testing.T) {
	min := 100.0
	credited := false
	repo := &mockRepo{
		getProgramFn: func(_ context.Context, _ string) (*EarnBurnProgram, error) {
			return &EarnBurnProgram{CustomerSisfiID: "cs-1", PointsRatio: 10, MinTicketAmount: &min}, nil
		},
		addPointsTxFn: func(_ context.Context, _ *Transaction) (int, error) {
			credited = true
			return 0, nil
		},
	}
	svc := newTestService(repo, newMockCache())

	_, err := svc.AddPoints(context.Background(), AddPointsReq{ClientID: "c1", CustomerSisfiID: "cs-1", Amount: 50})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ticket mínimo")
	assert.False(t, credited, "no debe acreditar puntos por debajo del mínimo")
}

func TestAddPoints_AtOrAboveMinTicket_Credited(t *testing.T) {
	min := 100.0
	repo := &mockRepo{
		getProgramFn: func(_ context.Context, _ string) (*EarnBurnProgram, error) {
			return &EarnBurnProgram{CustomerSisfiID: "cs-1", PointsRatio: 10, MinTicketAmount: &min}, nil
		},
		addPointsTxFn: func(_ context.Context, tx *Transaction) (int, error) { return tx.Amount, nil },
	}
	svc := newTestService(repo, newMockCache())

	tx, err := svc.AddPoints(context.Background(), AddPointsReq{ClientID: "c1", CustomerSisfiID: "cs-1", Amount: 100})

	require.NoError(t, err)
	assert.Equal(t, 10, tx.Amount) // 100 / 10
}

func TestAddPoints_NoMinTicket_UnchangedBehavior(t *testing.T) {
	repo := &mockRepo{
		getProgramFn: func(_ context.Context, _ string) (*EarnBurnProgram, error) {
			return &EarnBurnProgram{CustomerSisfiID: "cs-1", PointsRatio: 10}, nil // MinTicketAmount nil
		},
		addPointsTxFn: func(_ context.Context, tx *Transaction) (int, error) { return tx.Amount, nil },
	}
	svc := newTestService(repo, newMockCache())

	tx, err := svc.AddPoints(context.Background(), AddPointsReq{ClientID: "c1", CustomerSisfiID: "cs-1", Amount: 30})

	require.NoError(t, err)
	assert.Equal(t, 3, tx.Amount)
}

// --- FID-34: expiración lazy (earn_burn) ---

func TestCheckBalance_ExpiryConfigured_UsesExpireSweep(t *testing.T) {
	days := 30
	called := false
	repo := &mockRepo{
		getProgramFn: func(_ context.Context, _ string) (*EarnBurnProgram, error) {
			return &EarnBurnProgram{CustomerSisfiID: "cs-1", PointsRatio: 10, ExpiryDays: &days}, nil
		},
		expirePointsFn: func(_ context.Context, _, _ string, expiryDays int) (int, error) {
			called = true
			assert.Equal(t, 30, expiryDays)
			return 7, nil
		},
		getBalanceFn: func(_ context.Context, _, _ string) (int, error) { return 99, nil },
	}
	svc := newTestService(repo, newMockCache())

	bal, err := svc.CheckBalance(context.Background(), "c1", "cs-1")

	require.NoError(t, err)
	assert.True(t, called, "debe correr el sweep de expiración")
	assert.Equal(t, 7, bal, "debe usar el balance del sweep, no getBalance")
}

func TestCheckBalance_NoExpiry_UnchangedBehavior(t *testing.T) {
	repo := &mockRepo{
		getProgramFn: func(_ context.Context, _ string) (*EarnBurnProgram, error) {
			return &EarnBurnProgram{CustomerSisfiID: "cs-1", PointsRatio: 10}, nil // ExpiryDays nil
		},
		expirePointsFn: func(_ context.Context, _, _ string, _ int) (int, error) {
			t.Fatal("no debe expirar cuando expiry_days es nil")
			return 0, nil
		},
		getBalanceFn: func(_ context.Context, _, _ string) (int, error) { return 42, nil },
	}
	svc := newTestService(repo, newMockCache())

	bal, err := svc.CheckBalance(context.Background(), "c1", "cs-1")

	require.NoError(t, err)
	assert.Equal(t, 42, bal)
}
