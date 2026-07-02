package earnburn

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

// AddPoints calculates points from purchase amount and credits them.
func (s *Service) AddPoints(ctx context.Context, req AddPointsReq) (*Transaction, error) {
	program, err := s.repo.GetProgramByID(ctx, req.CustomerSisfiID)
	if err != nil {
		return nil, fmt.Errorf("get program: %w", err)
	}

	// FID-36: ticket mínimo. Si está configurado y la compra no lo alcanza,
	// no se acredita. nil = sin mínimo.
	if program.MinTicketAmount != nil && req.Amount < *program.MinTicketAmount {
		return nil, fmt.Errorf("monto de compra ($%.2f) menor al ticket mínimo ($%.2f); no se acreditan puntos", req.Amount, *program.MinTicketAmount)
	}

	points := int(math.Floor(req.Amount / float64(program.PointsRatio)))
	if points <= 0 {
		return nil, fmt.Errorf("monto insuficiente para acumular puntos (ratio: %d)", program.PointsRatio)
	}

	correctableUntil := time.Now().Add(correctionWindow)
	tx := &Transaction{
		ID:               generateUUID(),
		ClientID:         req.ClientID,
		CustomerSisfiID:  req.CustomerSisfiID,
		CollaboratorID:   req.CollaboratorID,
		Type:             "earn",
		Amount:           points,
		InvoiceURL:       req.InvoiceURL,
		ManualEntry:      req.ManualEntry,
		CorrectableUntil: &correctableUntil,
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

	newBalance, err := s.repo.AddPointsTx(ctx, tx)
	if err != nil {
		if errors.Is(err, ErrDuplicateReceipt) {
			return nil, apperror.Conflict("ticket ya registrado", err)
		}
		return nil, fmt.Errorf("add points tx: %w", err)
	}

	tx.BalanceAfter = newBalance

	s.log.Info("points.added",
		"client_id", req.ClientID,
		"amount", points,
		"balance_after", newBalance,
		"collaborator_id", req.CollaboratorID,
	)

	return tx, nil
}

// CheckBalance returns the current balance for a client.
//
// FID-34 (expiración lazy): si el programa tiene expiry_days configurado, antes
// de devolver el saldo se expiran de forma perezosa las cargas ("earn") cuyos
// puntos ya vencieron —registrando una transacción "expiration" compensatoria—.
// nil = sin vencimiento, comportamiento intacto.
func (s *Service) CheckBalance(ctx context.Context, clientID, customerSisfiID string) (int, error) {
	program, err := s.repo.GetProgramByID(ctx, customerSisfiID)
	if err == nil && program.ExpiryDays != nil {
		if balance, expErr := s.repo.ExpirePoints(ctx, clientID, customerSisfiID, *program.ExpiryDays); expErr == nil {
			return balance, nil
		} else {
			s.log.Error("expire points failed, returning stored balance", "error", expErr, "client_id", clientID)
		}
	}
	return s.repo.GetBalance(ctx, clientID, customerSisfiID)
}

// ListTransactions returns recent transactions.
func (s *Service) ListTransactions(ctx context.Context, clientID, customerSisfiID string, limit int) ([]Transaction, error) {
	return s.repo.ListTransactions(ctx, clientID, customerSisfiID, limit)
}

// ListCorrectableTransactions returns transactions within the 2h correction window.
func (s *Service) ListCorrectableTransactions(ctx context.Context, clientID string) ([]Transaction, error) {
	return s.repo.ListCorrectableTransactions(ctx, clientID)
}

// UpdatePoints applies a correction to an existing transaction.
func (s *Service) UpdatePoints(ctx context.Context, req UpdatePointsReq) (*Transaction, error) {
	original, err := s.repo.GetTransaction(ctx, req.TransactionID)
	if err != nil {
		return nil, fmt.Errorf("get original transaction: %w", err)
	}

	if original.CorrectableUntil == nil || time.Now().After(*original.CorrectableUntil) {
		return nil, fmt.Errorf("ventana de correccion expirada")
	}

	delta := req.NewAmount - original.Amount

	tx := &Transaction{
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

	newBalance, err := s.repo.AdjustPointsTx(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("adjust points: %w", err)
	}

	tx.BalanceAfter = newBalance

	s.log.Info("points.adjusted",
		"client_id", original.ClientID,
		"original_tx", req.TransactionID,
		"delta", delta,
		"balance_after", newBalance,
	)

	return tx, nil
}

// ListRewards returns rewards the client can afford.
func (s *Service) ListRewards(ctx context.Context, customerID, customerSisfiID string, maxPoints int) ([]Reward, error) {
	return s.repo.ListRewards(ctx, customerID, customerSisfiID, maxPoints)
}

// RequestRedemption creates a pending redemption with a temporary code.
func (s *Service) RequestRedemption(ctx context.Context, req RedemptionReq) (*Redemption, string, error) {
	reward, err := s.repo.GetReward(ctx, req.RewardID)
	if err != nil {
		return nil, "", fmt.Errorf("get reward: %w", err)
	}

	balance, err := s.repo.GetBalance(ctx, req.ClientID, req.CustomerSisfiID)
	if err != nil {
		return nil, "", fmt.Errorf("get balance: %w", err)
	}

	if balance < reward.PointsCost {
		return nil, "", fmt.Errorf("puntos insuficientes: tienes %d, necesitas %d", balance, reward.PointsCost)
	}

	code := generateCode(otpCodeLength)
	rd := &Redemption{
		ID:              generateUUID(),
		ClientID:        req.ClientID,
		RewardID:        req.RewardID,
		CustomerSisfiID: req.CustomerSisfiID,
		Code:            code,
		Status:          "pending",
		PointsSpent:     reward.PointsCost,
		ExpiresAt:       time.Now().Add(redemptionExpiry),
	}

	tx := &Transaction{
		ID:              generateUUID(),
		ClientID:        req.ClientID,
		CustomerSisfiID: req.CustomerSisfiID,
		Type:            "burn",
		Amount:          -reward.PointsCost,
		Description:     fmt.Sprintf("Canje: %s", reward.Name),
	}

	if err := s.repo.BurnPointsTx(ctx, tx, rd); err != nil {
		return nil, "", fmt.Errorf("burn points: %w", err)
	}

	// Store OTP in Redis for fast lookup
	otpData := &OTPData{
		ClientID:   req.ClientID,
		CustomerID: reward.CustomerID,
		Type:       "redemption",
		Metadata: map[string]string{
			"reward_id":    req.RewardID,
			"reward_name":  reward.Name,
			"points_spent": fmt.Sprintf("%d", reward.PointsCost),
		},
	}
	if err := s.cache.SetOTP(ctx, code, otpData); err != nil {
		s.log.Error("failed to cache redemption otp", "error", err)
		// Non-fatal: Postgres is source of truth
	}

	s.log.Info("redemption.requested",
		"client_id", req.ClientID,
		"reward", reward.Name,
		"code", code,
	)

	return rd, code, nil
}

// ConfirmRedemption validates the code and marks the redemption as confirmed.
func (s *Service) ConfirmRedemption(ctx context.Context, code, collaboratorID string) (*Redemption, error) {
	// Try Redis first (fast path)
	otpData, err := s.cache.ConsumeOTP(ctx, code)
	if err != nil {
		s.log.Error("redis consume failed, falling back to postgres", "error", err, "code", code)
	}

	if otpData != nil && otpData.Type != "redemption" {
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

	s.log.Info("redemption.confirmed",
		"code", code,
		"collaborator_id", collaboratorID,
	)

	return rd, nil
}

// RequestLoadPointsCode generates a temporary code for client->collaborator handoff.
func (s *Service) RequestLoadPointsCode(ctx context.Context, clientID, customerID string) (string, error) {
	code := generateCode(otpCodeLength)
	otpData := &OTPData{
		ClientID:   clientID,
		CustomerID: customerID,
		Type:       "load_points",
		Metadata:   map[string]string{},
	}
	if err := s.cache.SetOTP(ctx, code, otpData); err != nil {
		return "", fmt.Errorf("set load points otp: %w", err)
	}
	return code, nil
}

// ValidateLoadPointsCode checks and consumes the load points code.
func (s *Service) ValidateLoadPointsCode(ctx context.Context, code string) (*OTPData, error) {
	data, err := s.cache.ConsumeOTP(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("consume load points code [code=%s]: %w", code, err)
	}
	if data == nil {
		return nil, fmt.Errorf("codigo invalido o expirado [code=%s]", code)
	}
	if data.Type != "load_points" {
		return nil, fmt.Errorf("codigo invalido (tipo incorrecto) [code=%s, got_type=%s]", code, data.Type)
	}
	return data, nil
}

// RequestIdentityOTP generates an identity OTP for a client.
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
		Type:       "identity",
		Metadata:   map[string]string{},
	}
	if err := s.cache.SetOTP(ctx, code, otpData); err != nil {
		return "", fmt.Errorf("set identity otp: %w", err)
	}
	if err := s.cache.SetActiveIdentity(ctx, clientID, code); err != nil {
		s.log.Error("failed to set active identity tracker", "error", err)
	}
	return code, nil
}

// ValidateIdentityOTP checks an identity OTP (multi-use, does not consume).
func (s *Service) ValidateIdentityOTP(ctx context.Context, code string) (*OTPData, error) {
	data, err := s.cache.GetOTP(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("get identity otp [code=%s]: %w", code, err)
	}
	if data == nil {
		return nil, fmt.Errorf("codigo invalido o expirado [code=%s]", code)
	}
	if data.Type != "identity" {
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

func (s *Service) GetProgram(ctx context.Context, customerID string) (*EarnBurnProgram, error) {
	p, err := s.repo.GetProgram(ctx, customerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.NotFound("programa no encontrado", err)
		}
		return nil, apperror.Internal("error al buscar programa", err)
	}
	return p, nil
}

func (s *Service) GetReward(ctx context.Context, id string) (*Reward, error) {
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

func (s *Service) ListPrograms(ctx context.Context, customerID string) ([]EarnBurnProgram, error) {
	programs, err := s.repo.ListPrograms(ctx, customerID)
	if err != nil {
		return nil, apperror.Internal("error al listar programas", err)
	}
	return programs, nil
}

func (s *Service) CreateProgram(ctx context.Context, p *EarnBurnProgram) error {
	if p.CustomerID == "" || p.Name == "" || p.PointsRatio <= 0 {
		return apperror.BadRequest("customer_id, name y points_ratio (>0) son requeridos", nil)
	}
	if err := s.repo.CreateProgram(ctx, p); err != nil {
		return apperror.Internal("error al crear programa", err)
	}
	return nil
}

// UpdateProgram updates an existing program's name, points_ratio and the
// loyalty config options (expiry_days, min_ticket_amount). setActive toggles the
// program active flag; nil leaves it unchanged.
func (s *Service) UpdateProgram(ctx context.Context, p *EarnBurnProgram, setActive *bool) error {
	if p.CustomerSisfiID == "" {
		return apperror.BadRequest("id de programa requerido", nil)
	}
	if p.ExpiryDays != nil && *p.ExpiryDays <= 0 {
		return apperror.BadRequest("expiry_days debe ser mayor a 0", nil)
	}
	if p.MinTicketAmount != nil && *p.MinTicketAmount < 0 {
		return apperror.BadRequest("min_ticket_amount no puede ser negativo", nil)
	}
	if err := s.repo.UpdateProgram(ctx, p, setActive); err != nil {
		return apperror.Internal("error al actualizar programa", err)
	}
	return nil
}

func (s *Service) GetCustomer(ctx context.Context, id string) (*Customer, error) {
	c, err := s.repo.GetCustomer(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.NotFound("cliente no encontrado", err)
		}
		return nil, apperror.Internal("error al buscar cliente", err)
	}
	return c, nil
}

func (s *Service) CreateCustomer(ctx context.Context, c *Customer) error {
	if err := s.repo.CreateCustomer(ctx, c); err != nil {
		return apperror.Internal("error al crear cliente", err)
	}
	return nil
}

func (s *Service) UpdateCustomer(ctx context.Context, c *Customer) error {
	if err := s.repo.UpdateCustomer(ctx, c); err != nil {
		return apperror.Internal("error al actualizar cliente", err)
	}
	return nil
}

func (s *Service) CreateCollaborator(ctx context.Context, c *Collaborator) error {
	if err := s.repo.CreateCollaborator(ctx, c); err != nil {
		return apperror.Internal("error al crear colaborador", err)
	}
	return nil
}

func (s *Service) ListCollaborators(ctx context.Context, customerID string) ([]Collaborator, error) {
	collabs, err := s.repo.ListCollaborators(ctx, customerID)
	if err != nil {
		return nil, apperror.Internal("error al listar colaboradores", err)
	}
	return collabs, nil
}

func (s *Service) ListAllRewards(ctx context.Context, customerSisfiID string) ([]Reward, error) {
	rewards, err := s.repo.ListAllRewards(ctx, customerSisfiID)
	if err != nil {
		return nil, apperror.Internal("error al listar recompensas", err)
	}
	return rewards, nil
}

func (s *Service) CreateRewardAdmin(ctx context.Context, customerSisfiID string, r *Reward) error {
	if err := s.repo.CreateRewardAdmin(ctx, customerSisfiID, r); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperror.NotFound("programa no encontrado", err)
		}
		return apperror.Internal("error al crear recompensa", err)
	}
	return nil
}

func (s *Service) UpdateRewardAdmin(ctx context.Context, r *Reward) error {
	if err := s.repo.UpdateRewardAdmin(ctx, r); err != nil {
		return apperror.Internal("error al actualizar recompensa", err)
	}
	return nil
}

func (s *Service) ListClients(ctx context.Context, customerID string) ([]Client, error) {
	clients, err := s.repo.ListClients(ctx, customerID)
	if err != nil {
		return nil, apperror.Internal("error al listar clientes", err)
	}
	return clients, nil
}

func (s *Service) ListFeedback(ctx context.Context, customerID string) ([]FeedbackEntry, error) {
	entries, err := s.repo.ListFeedback(ctx, customerID)
	if err != nil {
		return nil, apperror.Internal("error al listar feedback", err)
	}
	return entries, nil
}

func (s *Service) GetBalance(ctx context.Context, clientID, customerSisfiID string) (int, error) {
	return s.repo.GetBalance(ctx, clientID, customerSisfiID)
}

// GetCustomerMetrics agrega y calcula las métricas T1 del dashboard para un customer.
func (s *Service) GetCustomerMetrics(ctx context.Context, customerID string) (*CustomerMetrics, error) {
	if customerID == "" {
		return nil, apperror.BadRequest("customer_id requerido", nil)
	}
	raw, err := s.repo.GetMetricsRaw(ctx, customerID)
	if err != nil {
		return nil, apperror.Internal("error al calcular métricas", err)
	}
	metrics := ComputeMetrics(raw)
	return &metrics, nil
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
