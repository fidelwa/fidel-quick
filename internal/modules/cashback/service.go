package cashback

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"time"

	"github.com/theluisbolivar/fidel-quick/internal/apperror"
)

const (
	otpCodeLength    = 6
	correctionWindow = 2 * time.Hour
	redemptionExpiry = 1 * time.Hour
)

type Service struct {
	repo  Repository
	cache Cache
	log   *slog.Logger
}

func NewService(repo Repository, cache Cache, log *slog.Logger) *Service {
	return &Service{repo: repo, cache: cache, log: log}
}

// AddCashback calculates cashback from purchase amount and credits it.
func (s *Service) AddCashback(ctx context.Context, req AddCashbackReq) (*CashbackTransaction, error) {
	program, err := s.repo.GetProgramByID(ctx, req.ProgramID)
	if err != nil {
		return nil, fmt.Errorf("get cashback program: %w", err)
	}

	cashbackAmount := math.Floor(req.Amount*program.CashbackRate*100) / 100
	if cashbackAmount <= 0 {
		return nil, fmt.Errorf("monto insuficiente para acumular cashback (rate: %.2f%%)", program.CashbackRate*100)
	}

	correctableUntil := time.Now().Add(correctionWindow)
	tx := &CashbackTransaction{
		ID:               generateUUID(),
		ClientID:         req.ClientID,
		ProgramID:        req.ProgramID,
		CollaboratorID:   req.CollaboratorID,
		Type:             "earn",
		Amount:           cashbackAmount,
		PurchaseAmount:   req.Amount,
		InvoiceURL:       req.InvoiceURL,
		ManualEntry:      req.ManualEntry,
		CorrectableUntil: &correctableUntil,
	}

	newBalance, err := s.repo.AddCashbackTx(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("add cashback tx: %w", err)
	}

	tx.BalanceAfter = newBalance

	s.log.Info("cashback.added",
		"client_id", req.ClientID,
		"purchase_amount", req.Amount,
		"cashback_amount", cashbackAmount,
		"balance_after", newBalance,
		"collaborator_id", req.CollaboratorID,
	)

	return tx, nil
}

// CheckBalance returns the current cashback balance for a client.
func (s *Service) CheckBalance(ctx context.Context, clientID, programID string) (float64, error) {
	return s.repo.GetBalance(ctx, clientID, programID)
}

// ListTransactions returns recent cashback transactions.
func (s *Service) ListTransactions(ctx context.Context, clientID, programID string, limit int) ([]CashbackTransaction, error) {
	return s.repo.ListTransactions(ctx, clientID, programID, limit)
}

// ListCorrectableTransactions returns transactions within the 2h correction window.
func (s *Service) ListCorrectableTransactions(ctx context.Context, clientID string) ([]CashbackTransaction, error) {
	return s.repo.ListCorrectableTransactions(ctx, clientID)
}

// UpdateCashback applies a correction: receives new invoice amount, recalculates cashback, applies delta.
func (s *Service) UpdateCashback(ctx context.Context, req UpdateCashbackReq) (*CashbackTransaction, error) {
	original, err := s.repo.GetTransaction(ctx, req.TransactionID)
	if err != nil {
		return nil, fmt.Errorf("get original transaction: %w", err)
	}

	if original.CorrectableUntil == nil || time.Now().After(*original.CorrectableUntil) {
		return nil, fmt.Errorf("ventana de correccion expirada")
	}

	// Get program to recalculate cashback
	program, err := s.repo.GetProgramByID(ctx, original.ProgramID)
	if err != nil {
		return nil, fmt.Errorf("get program for correction: %w", err)
	}

	// Recalculate cashback from new purchase amount
	newCashback := math.Floor(req.NewPurchaseAmount*program.CashbackRate*100) / 100
	delta := newCashback - original.Amount

	tx := &CashbackTransaction{
		ID:                    generateUUID(),
		ClientID:              original.ClientID,
		ProgramID:             original.ProgramID,
		CollaboratorID:        req.CollaboratorID,
		Type:                  "adjustment",
		Amount:                delta,
		Description:           req.TransactionID, // reference to original
		CorrectionReason:      req.CorrectionReason,
		CorrectionEvidenceURL: req.CorrectionEvidenceURL,
	}

	newBalance, err := s.repo.AdjustCashbackTx(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("adjust cashback: %w", err)
	}

	tx.BalanceAfter = newBalance

	s.log.Info("cashback.adjusted",
		"client_id", original.ClientID,
		"original_tx", req.TransactionID,
		"delta", delta,
		"balance_after", newBalance,
	)

	return tx, nil
}

// ListRewards returns rewards the client can afford.
func (s *Service) ListRewards(ctx context.Context, customerID, programID string, maxCost float64) ([]CashbackReward, error) {
	return s.repo.ListRewards(ctx, customerID, programID, maxCost)
}

// RequestRedemption creates a pending redemption with a temporary code.
func (s *Service) RequestRedemption(ctx context.Context, req CashbackRedemptionReq) (*CashbackRedemption, string, error) {
	reward, err := s.repo.GetReward(ctx, req.RewardID)
	if err != nil {
		return nil, "", fmt.Errorf("get reward: %w", err)
	}

	balance, err := s.repo.GetBalance(ctx, req.ClientID, req.ProgramID)
	if err != nil {
		return nil, "", fmt.Errorf("get balance: %w", err)
	}

	if balance < reward.Cost {
		return nil, "", fmt.Errorf("saldo insuficiente: tienes $%.2f, necesitas $%.2f", balance, reward.Cost)
	}

	code := generateCode(otpCodeLength)
	rd := &CashbackRedemption{
		ID:          generateUUID(),
		ClientID:    req.ClientID,
		RewardID:    req.RewardID,
		ProgramID:   req.ProgramID,
		Code:        code,
		Status:      "pending",
		AmountSpent: reward.Cost,
		ExpiresAt:   time.Now().Add(redemptionExpiry),
	}

	tx := &CashbackTransaction{
		ID:          generateUUID(),
		ClientID:    req.ClientID,
		ProgramID:   req.ProgramID,
		Type:        "burn",
		Amount:      -reward.Cost,
		Description: fmt.Sprintf("Canje: %s", reward.Name),
	}

	if err := s.repo.BurnCashbackTx(ctx, tx, rd); err != nil {
		return nil, "", fmt.Errorf("burn cashback: %w", err)
	}

	// Store OTP in Redis for fast lookup
	otpData := &OTPData{
		ClientID:   req.ClientID,
		CustomerID: reward.CustomerID,
		Type:       "cb_redemption",
		Metadata: map[string]string{
			"reward_id":    req.RewardID,
			"reward_name":  reward.Name,
			"amount_spent": fmt.Sprintf("%.2f", reward.Cost),
		},
	}
	if err := s.cache.SetOTP(ctx, code, otpData); err != nil {
		s.log.Error("failed to cache cashback redemption otp", "error", err)
	}

	s.log.Info("cashback.redemption.requested",
		"client_id", req.ClientID,
		"reward", reward.Name,
		"code", code,
	)

	return rd, code, nil
}

// ConfirmRedemption validates the code and marks the redemption as confirmed.
func (s *Service) ConfirmRedemption(ctx context.Context, code, collaboratorID string) (*CashbackRedemption, error) {
	// Try Redis first (fast path)
	otpData, err := s.cache.ConsumeOTP(ctx, code)
	if err != nil {
		s.log.Error("redis consume failed, falling back to postgres", "error", err, "code", code)
	}

	if otpData != nil && otpData.Type != "cb_redemption" {
		return nil, fmt.Errorf("codigo invalido (tipo incorrecto) [code=%s, got_type=%s]", code, otpData.Type)
	}

	// Always confirm via Postgres (source of truth)
	rd, err := s.repo.GetRedemptionByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("codigo de canje no encontrado [code=%s]", code)
	}

	if rd.Status != "pending" {
		return nil, fmt.Errorf("canje ya fue %s", rd.Status)
	}

	if time.Now().After(rd.ExpiresAt) {
		return nil, fmt.Errorf("codigo expirado")
	}

	if err := s.repo.ConfirmRedemption(ctx, rd.ID, collaboratorID); err != nil {
		return nil, fmt.Errorf("confirm redemption: %w", err)
	}

	rd.Status = "confirmed"
	rd.ConfirmedBy = collaboratorID

	s.log.Info("cashback.redemption.confirmed",
		"code", code,
		"collaborator_id", collaboratorID,
	)

	return rd, nil
}

// RequestLoadCode generates a cashback identity code (multi-use, 15 min).
// This single code is used by all collaborator flows.
func (s *Service) RequestLoadCode(ctx context.Context, clientID, customerID string) (string, error) {
	return s.RequestIdentityOTP(ctx, clientID, customerID)
}

// ValidateLoadCode checks a cashback identity code (multi-use, does not consume).
func (s *Service) ValidateLoadCode(ctx context.Context, code string) (*OTPData, error) {
	return s.ValidateIdentityOTP(ctx, code)
}

// RequestIdentityOTP generates a cashback-specific identity code for a client.
func (s *Service) RequestIdentityOTP(ctx context.Context, clientID, customerID string) (string, error) {
	// Invalidate previous OTP
	oldCode, _ := s.cache.GetActiveIdentity(ctx, clientID)
	if oldCode != "" {
		s.cache.DeleteOTP(ctx, oldCode)
		s.cache.DeleteActiveIdentity(ctx, clientID)
	}

	code := generateCode(otpCodeLength)
	otpData := &OTPData{
		ClientID:   clientID,
		CustomerID: customerID,
		Type:       "cb_identity",
		Metadata:   map[string]string{},
	}
	if err := s.cache.SetOTP(ctx, code, otpData); err != nil {
		return "", fmt.Errorf("set cb identity otp: %w", err)
	}
	if err := s.cache.SetActiveIdentity(ctx, clientID, code); err != nil {
		s.log.Error("failed to set cb active identity tracker", "error", err)
	}
	return code, nil
}

// ValidateIdentityOTP checks a cashback identity OTP (multi-use, does not consume).
func (s *Service) ValidateIdentityOTP(ctx context.Context, code string) (*OTPData, error) {
	data, err := s.cache.GetOTP(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("get cb identity otp [code=%s]: %w", code, err)
	}
	if data == nil {
		return nil, fmt.Errorf("codigo invalido o expirado [code=%s]", code)
	}
	if data.Type != "cb_identity" {
		return nil, fmt.Errorf("codigo invalido (tipo incorrecto) [code=%s, got_type=%s]", code, data.Type)
	}
	return data, nil
}

// GetClientName returns the client's name.
func (s *Service) GetClientName(ctx context.Context, clientID string) (string, error) {
	return s.repo.GetClientName(ctx, clientID)
}

// SubmitFeedback stores client feedback.
func (s *Service) SubmitFeedback(ctx context.Context, clientID, customerID, message string) error {
	return s.repo.CreateFeedback(ctx, clientID, customerID, message)
}

// --- Passthroughs for module.go ---

func (s *Service) GetProgram(ctx context.Context, customerID string) (*CashbackProgram, error) {
	p, err := s.repo.GetProgram(ctx, customerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.NotFound("programa cashback no encontrado", err)
		}
		return nil, apperror.Internal("error al buscar programa cashback", err)
	}
	return p, nil
}

func (s *Service) GetReward(ctx context.Context, id string) (*CashbackReward, error) {
	r, err := s.repo.GetReward(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.NotFound("beneficio no encontrado", err)
		}
		return nil, apperror.Internal("error al buscar beneficio", err)
	}
	return r, nil
}

// --- Admin CRUD ---

func (s *Service) ListPrograms(ctx context.Context, customerID string) ([]CashbackProgram, error) {
	programs, err := s.repo.ListPrograms(ctx, customerID)
	if err != nil {
		return nil, apperror.Internal("error al listar programas cashback", err)
	}
	return programs, nil
}

func (s *Service) CreateProgram(ctx context.Context, p *CashbackProgram) error {
	if err := s.repo.CreateProgram(ctx, p); err != nil {
		return apperror.Internal("error al crear programa cashback", err)
	}
	return nil
}

func (s *Service) UpdateProgram(ctx context.Context, p *CashbackProgram) error {
	if err := s.repo.UpdateProgram(ctx, p); err != nil {
		return apperror.Internal("error al actualizar programa cashback", err)
	}
	return nil
}

func (s *Service) ListAllRewards(ctx context.Context, programID string) ([]CashbackReward, error) {
	rewards, err := s.repo.ListAllRewards(ctx, programID)
	if err != nil {
		return nil, apperror.Internal("error al listar beneficios", err)
	}
	return rewards, nil
}

func (s *Service) CreateRewardAdmin(ctx context.Context, programID string, r *CashbackReward) error {
	if err := s.repo.CreateRewardAdmin(ctx, programID, r); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperror.NotFound("programa cashback no encontrado", err)
		}
		return apperror.Internal("error al crear beneficio", err)
	}
	return nil
}

func (s *Service) UpdateRewardAdmin(ctx context.Context, r *CashbackReward) error {
	if err := s.repo.UpdateRewardAdmin(ctx, r); err != nil {
		return apperror.Internal("error al actualizar beneficio", err)
	}
	return nil
}

func (s *Service) GetBalance(ctx context.Context, clientID, programID string) (float64, error) {
	return s.repo.GetBalance(ctx, clientID, programID)
}

// generateCode creates a random alphanumeric code.
func generateCode(length int) string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no I, O, 0, 1
	code := make([]byte, length)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		code[i] = chars[n.Int64()]
	}
	return string(code)
}

// generateUUID creates a v4 UUID string.
func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
