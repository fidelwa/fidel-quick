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
	"github.com/theluisbolivar/fidel-quick/internal/platform/ai"
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
	program, err := s.repo.GetProgramByID(ctx, req.CustomerSisfiID)
	if err != nil {
		return nil, fmt.Errorf("get cashback program: %w", err)
	}

	// FID-36: ticket mínimo. nil = sin mínimo.
	if program.MinTicketAmount != nil && req.Amount < *program.MinTicketAmount {
		return nil, fmt.Errorf("monto de compra ($%.2f) menor al ticket mínimo ($%.2f); no se acredita cashback", req.Amount, *program.MinTicketAmount)
	}

	cashbackAmount := math.Floor(req.Amount*program.CashbackRate*100) / 100
	if cashbackAmount <= 0 {
		return nil, fmt.Errorf("monto insuficiente para acumular cashback (rate: %.2f%%)", program.CashbackRate*100)
	}

	// FID-37: caps de cashback. nil = sin cap.
	// Techo por transacción (depende solo del monto de la compra, sin carrera).
	if program.MaxCashbackPerTx != nil && cashbackAmount > *program.MaxCashbackPerTx {
		cashbackAmount = *program.MaxCashbackPerTx
	}

	if cashbackAmount <= 0 {
		return nil, fmt.Errorf("monto insuficiente para acumular cashback tras aplicar límites")
	}

	correctableUntil := time.Now().Add(correctionWindow)
	tx := &CashbackTransaction{
		ID:               generateUUID(),
		ClientID:         req.ClientID,
		CustomerSisfiID:  req.CustomerSisfiID,
		CollaboratorID:   req.CollaboratorID,
		Type:             "earn",
		Amount:           cashbackAmount,
		PurchaseAmount:   req.Amount,
		InvoiceURL:       req.InvoiceURL,
		ManualEntry:      req.ManualEntry,
		CorrectableUntil: &correctableUntil,
	}

	// FID-37 (LG-2): techo por periodo aplicado de forma ATÓMICA dentro de
	// AddCashbackTx. Pasamos el cap y la ventana (expiry_days o 30 días por
	// defecto); la lectura de la suma acumulada y el clamp ocurren en la misma
	// transacción que el insert, evitando la carrera de requests concurrentes.
	if program.MaxCashbackPerPeriod != nil {
		windowDays := 30
		if program.ExpiryDays != nil {
			windowDays = *program.ExpiryDays
		}
		tx.PeriodCap = program.MaxCashbackPerPeriod
		tx.PeriodWindowDays = windowDays
	}

	// Anti-fraude (FID-33): persistir el extract y calcular el hash de dedup.
	if req.Invoice != nil {
		fp, err := ai.ComputeFingerprint(req.Invoice)
		if err != nil {
			return nil, fmt.Errorf("compute receipt fingerprint: %w", err)
		}
		tx.ReceiptData = fp.Data
		tx.ReceiptHash = fp.Hash
		tx.ReceiptHashFields = fp.HashFields
		tx.ReceiptConfident = fp.Confident
	}

	newBalance, err := s.repo.AddCashbackTx(ctx, tx)
	if err != nil {
		if errors.Is(err, ErrDuplicateReceipt) {
			return nil, apperror.Conflict("ticket ya registrado", err)
		}
		if errors.Is(err, ErrPeriodCapExhausted) {
			return nil, fmt.Errorf("se alcanzó el máximo de cashback del periodo ($%.2f)", *program.MaxCashbackPerPeriod)
		}
		return nil, fmt.Errorf("add cashback tx: %w", err)
	}

	// El monto pudo recortarse dentro de la tx por el cap por periodo.
	cashbackAmount = tx.Amount

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
//
// FID-34 (expiración lazy): si el programa tiene expiry_days configurado, expira
// de forma perezosa las acreditaciones vencidas antes de devolver el saldo.
// nil = sin vencimiento, comportamiento intacto.
func (s *Service) CheckBalance(ctx context.Context, clientID, customerSisfiID string) (float64, error) {
	program, err := s.repo.GetProgramByID(ctx, customerSisfiID)
	if err == nil && program.ExpiryDays != nil {
		if balance, expErr := s.repo.ExpireBalance(ctx, clientID, customerSisfiID, *program.ExpiryDays); expErr == nil {
			return balance, nil
		} else {
			s.log.Error("expire cashback failed, returning stored balance", "error", expErr, "client_id", clientID)
		}
	}
	return s.repo.GetBalance(ctx, clientID, customerSisfiID)
}

// ListTransactions returns recent cashback transactions.
func (s *Service) ListTransactions(ctx context.Context, clientID, customerSisfiID string, limit int) ([]CashbackTransaction, error) {
	return s.repo.ListTransactions(ctx, clientID, customerSisfiID, limit)
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
	program, err := s.repo.GetProgramByID(ctx, original.CustomerSisfiID)
	if err != nil {
		return nil, fmt.Errorf("get program for correction: %w", err)
	}

	// Recalculate cashback from new purchase amount
	newCashback := math.Floor(req.NewPurchaseAmount*program.CashbackRate*100) / 100
	delta := newCashback - original.Amount

	tx := &CashbackTransaction{
		ID:                    generateUUID(),
		ClientID:              original.ClientID,
		CustomerSisfiID:       original.CustomerSisfiID,
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
func (s *Service) ListRewards(ctx context.Context, customerID, customerSisfiID string, maxCost float64) ([]CashbackReward, error) {
	return s.repo.ListRewards(ctx, customerID, customerSisfiID, maxCost)
}

// RequestRedemption creates a pending redemption with a temporary code.
func (s *Service) RequestRedemption(ctx context.Context, req CashbackRedemptionReq) (*CashbackRedemption, string, error) {
	reward, err := s.repo.GetReward(ctx, req.RewardID)
	if err != nil {
		return nil, "", fmt.Errorf("get reward: %w", err)
	}

	balance, err := s.repo.GetBalance(ctx, req.ClientID, req.CustomerSisfiID)
	if err != nil {
		return nil, "", fmt.Errorf("get balance: %w", err)
	}

	if balance < reward.Cost {
		return nil, "", fmt.Errorf("saldo insuficiente: tienes $%.2f, necesitas $%.2f", balance, reward.Cost)
	}

	code := generateCode(otpCodeLength)
	rd := &CashbackRedemption{
		ID:              generateUUID(),
		ClientID:        req.ClientID,
		RewardID:        req.RewardID,
		CustomerSisfiID: req.CustomerSisfiID,
		Code:            code,
		Status:          "pending",
		AmountSpent:     reward.Cost,
		ExpiresAt:       time.Now().Add(redemptionExpiry),
	}

	tx := &CashbackTransaction{
		ID:              generateUUID(),
		ClientID:        req.ClientID,
		CustomerSisfiID: req.CustomerSisfiID,
		Type:            "burn",
		Amount:          -reward.Cost,
		Description:     fmt.Sprintf("Canje: %s", reward.Name),
	}

	if err := s.repo.BurnCashbackTx(ctx, tx, rd); err != nil {
		// FID-38: premio agotado. El burn se revirtió completo (no se gastó saldo
		// ni se creó canje). Error tipado para respuesta clara al cliente.
		if errors.Is(err, ErrRewardOutOfStock) {
			return nil, "", apperror.Conflict("premio agotado", err)
		}
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

func (s *Service) GetClientPhone(ctx context.Context, clientID string) (string, error) {
	return s.repo.GetClientPhone(ctx, clientID)
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
			return nil, apperror.NotFound("recompensa no encontrada", err)
		}
		return nil, apperror.Internal("error al buscar recompensa", err)
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

// CreateProgram activates cashback for a customer. Accepts cashback_rate either as
// a percentage (e.g. 5 for 5%) or as a fraction (e.g. 0.05); both are normalized to
// the fraction stored in config_cashback (DECIMAL(5,4) CHECK > 0 AND <= 1).
func (s *Service) CreateProgram(ctx context.Context, p *CashbackProgram) error {
	if p.CustomerID == "" || p.Name == "" || p.CashbackRate <= 0 {
		return apperror.BadRequest("customer_id, name y cashback_rate (>0) son requeridos", nil)
	}
	if p.CashbackRate > 1 {
		p.CashbackRate = p.CashbackRate / 100
	}
	if p.CashbackRate > 1 || p.CashbackRate <= 0 {
		return apperror.BadRequest("cashback_rate debe estar entre 0 y 1 (o entre 0 y 100 si es porcentaje)", nil)
	}
	if err := s.repo.CreateProgram(ctx, p); err != nil {
		return apperror.Internal("error al crear programa cashback", err)
	}
	return nil
}

// UpdateProgram updates an existing cashback program's name, cashback_rate and
// the loyalty config options (expiry_days, min_ticket_amount, max_cashback_per_tx,
// max_cashback_per_period). cashback_rate se normaliza igual que en CreateProgram
// (acepta porcentaje 0-100 o fracción 0-1). setActive nil deja el flag intacto.
func (s *Service) UpdateProgram(ctx context.Context, p *CashbackProgram, setActive *bool) error {
	if p.CustomerSisfiID == "" {
		return apperror.BadRequest("id de programa requerido", nil)
	}
	if p.CashbackRate > 1 {
		p.CashbackRate = p.CashbackRate / 100
	}
	if p.CashbackRate < 0 || p.CashbackRate > 1 {
		return apperror.BadRequest("cashback_rate debe estar entre 0 y 1 (o entre 0 y 100 si es porcentaje)", nil)
	}
	if p.ExpiryDays != nil && *p.ExpiryDays <= 0 {
		return apperror.BadRequest("expiry_days debe ser mayor a 0", nil)
	}
	if p.MinTicketAmount != nil && *p.MinTicketAmount < 0 {
		return apperror.BadRequest("min_ticket_amount no puede ser negativo", nil)
	}
	// FID-37 (LG-4): un cap de 0 envenenaría el programa (bloquearía todo el
	// cashback). "Sin cap" se representa con null (campo vacío en el form), no
	// con 0. Por eso un cap no-nil debe ser estrictamente > 0.
	if p.MaxCashbackPerTx != nil && *p.MaxCashbackPerTx <= 0 {
		return apperror.BadRequest("max_cashback_per_tx debe ser mayor a 0 (vacío = sin límite)", nil)
	}
	if p.MaxCashbackPerPeriod != nil && *p.MaxCashbackPerPeriod <= 0 {
		return apperror.BadRequest("max_cashback_per_period debe ser mayor a 0 (vacío = sin límite)", nil)
	}
	if err := s.repo.UpdateProgram(ctx, p, setActive); err != nil {
		return apperror.Internal("error al actualizar programa cashback", err)
	}
	return nil
}

func (s *Service) ListAllRewards(ctx context.Context, customerSisfiID string) ([]CashbackReward, error) {
	rewards, err := s.repo.ListAllRewards(ctx, customerSisfiID)
	if err != nil {
		return nil, apperror.Internal("error al listar recompensas", err)
	}
	return rewards, nil
}

func (s *Service) CreateRewardAdmin(ctx context.Context, customerSisfiID string, r *CashbackReward) error {
	if err := s.repo.CreateRewardAdmin(ctx, customerSisfiID, r); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperror.NotFound("programa cashback no encontrado", err)
		}
		return apperror.Internal("error al crear recompensa", err)
	}
	return nil
}

func (s *Service) UpdateRewardAdmin(ctx context.Context, r *CashbackReward) error {
	if err := s.repo.UpdateRewardAdmin(ctx, r); err != nil {
		return apperror.Internal("error al actualizar recompensa", err)
	}
	return nil
}

func (s *Service) GetBalance(ctx context.Context, clientID, customerSisfiID string) (float64, error) {
	return s.repo.GetBalance(ctx, clientID, customerSisfiID)
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
