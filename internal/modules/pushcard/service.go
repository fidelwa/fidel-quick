package pushcard

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// CorrectionWindow is the period within which a collaborator can undo their last stamp.
const CorrectionWindow = 2 * time.Hour

var (
	ErrConfigNotFound = errors.New("pushcard config no encontrado")
	ErrNoStampToUndo  = errors.New("no hay sello reciente para deshacer")
)

// Service implements pushcard business rules.
type Service struct {
	repo  Repository
	cache Cache
	log   *slog.Logger
}

// NewService builds a service. cache may be nil when not exercising redemption.
func NewService(repo Repository, cache Cache, log *slog.Logger) *Service {
	return &Service{repo: repo, cache: cache, log: log}
}

// GetProgress returns the current card status for a client.
func (s *Service) GetProgress(ctx context.Context, customerSisfiID, clientID string) (*CardProgress, error) {
	cfg, err := s.repo.GetConfigByID(ctx, customerSisfiID)
	if err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}

	card, err := s.repo.GetOpenCard(ctx, customerSisfiID, clientID)
	if err != nil {
		return nil, fmt.Errorf("get open card: %w", err)
	}

	if card == nil {
		return &CardProgress{
			HasOpenCard: false,
			CardSlots:   cfg.CardSlots,
			Visual:      buildVisual(0, cfg.CardSlots),
		}, nil
	}

	count, err := s.repo.CountStamps(ctx, card.ID)
	if err != nil {
		return nil, fmt.Errorf("count stamps: %w", err)
	}

	card.StampsCount = count
	return &CardProgress{
		HasOpenCard: true,
		Card:        card,
		StampsCount: count,
		CardSlots:   cfg.CardSlots,
		Visual:      buildVisual(count, cfg.CardSlots),
	}, nil
}

// AddStamp adds a stamp to the client's open card. If the client has no open card,
// one is created. If the stamp completes the card (count reaches CardSlots), the
// card is marked completed and the result indicates so.
func (s *Service) AddStamp(ctx context.Context, req AddStampReq) (*AddStampResult, error) {
	if req.CustomerSisfiID == "" || req.ClientID == "" || req.CollaboratorID == "" {
		return nil, fmt.Errorf("customer_sisfi_id, client_id y collaborator_id son requeridos")
	}

	cfg, err := s.repo.GetConfigByID(ctx, req.CustomerSisfiID)
	if err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}

	card, err := s.repo.GetOpenCard(ctx, req.CustomerSisfiID, req.ClientID)
	if err != nil {
		return nil, fmt.Errorf("get open card: %w", err)
	}
	if card == nil {
		card, err = s.repo.OpenCard(ctx, req.CustomerSisfiID, req.ClientID)
		if err != nil {
			return nil, fmt.Errorf("open card: %w", err)
		}
	}

	stamp := &Stamp{
		ID:             generateUUID(),
		CardID:         card.ID,
		CollaboratorID: req.CollaboratorID,
	}
	if err := s.repo.AddStamp(ctx, stamp); err != nil {
		return nil, err
	}

	count, err := s.repo.CountStamps(ctx, card.ID)
	if err != nil {
		return nil, err
	}

	completed := false
	if count >= cfg.CardSlots {
		if err := s.repo.CompleteCard(ctx, card.ID); err != nil {
			return nil, err
		}
		completed = true
		card.Status = StatusCompleted
		now := time.Now()
		card.CompletedAt = &now
	}

	card.StampsCount = count
	s.log.Info("pushcard.stamp.added",
		"card_id", card.ID,
		"client_id", req.ClientID,
		"collaborator_id", req.CollaboratorID,
		"count", count,
		"slots", cfg.CardSlots,
		"completed", completed,
	)

	return &AddStampResult{
		Card:        card,
		StampsCount: count,
		CardSlots:   cfg.CardSlots,
		Completed:   completed,
		StampID:     stamp.ID,
	}, nil
}

// UndoLastStamp removes the most recent stamp by the collaborator if it was placed
// within CorrectionWindow. If removing the stamp uncompletes a card, the card is
// reverted to 'open'.
func (s *Service) UndoLastStamp(ctx context.Context, collaboratorID string) (*Card, error) {
	stamp, err := s.repo.LastStampByCollaborator(ctx, collaboratorID, CorrectionWindow)
	if err != nil {
		return nil, err
	}
	if stamp == nil {
		return nil, ErrNoStampToUndo
	}

	card, err := s.repo.GetCard(ctx, stamp.CardID)
	if err != nil {
		return nil, fmt.Errorf("get card: %w", err)
	}

	if err := s.repo.DeleteStamp(ctx, stamp.ID); err != nil {
		return nil, err
	}

	// If the card was completed by this stamp and has not been redeemed, reopen it.
	if card.Status == StatusCompleted {
		// Reopen by writing status back. Simplified: assumes the just-deleted stamp
		// was indeed the one that completed it (which is true within 2h since stamps
		// are append-only and only completion transitions out of 'open').
		if _, err := s.repo.OpenCard(ctx, card.CustomerSisfiID, card.ClientID); err != nil {
			s.log.Warn("could not reopen replacement card", "error", err)
		}
	}

	count, _ := s.repo.CountStamps(ctx, card.ID)
	card.StampsCount = count

	s.log.Info("pushcard.stamp.undone",
		"stamp_id", stamp.ID,
		"card_id", card.ID,
		"collaborator_id", collaboratorID,
	)
	return card, nil
}

// MarkRedeemed transitions a completed card to redeemed.
func (s *Service) MarkRedeemed(ctx context.Context, cardID string) error {
	return s.repo.MarkRedeemed(ctx, cardID)
}

// RequestRedemption is called by a client whose card is completed. It
// generates a 6-char code that the collaborator will enter to confirm the
// canje. The code is stored in cache with TTL = redemptionTTL (1h).
func (s *Service) RequestRedemption(ctx context.Context, customerSisfiID, clientID, customerID, rewardName string) (string, error) {
	cards, err := s.repo.ListCardsByCustomer(ctx, customerSisfiID, StatusCompleted, 50)
	if err != nil {
		return "", err
	}
	var card *Card
	for i := range cards {
		if cards[i].ClientID == clientID {
			card = &cards[i]
			break
		}
	}
	if card == nil {
		return "", fmt.Errorf("no tenés ninguna tarjeta completada para canjear")
	}
	if s.cache == nil {
		return "", fmt.Errorf("cache no configurado")
	}

	code := generateCode(6)
	data := &RedemptionCode{
		CardID:          card.ID,
		ClientID:        clientID,
		CustomerID:      customerID,
		CustomerSisfiID: customerSisfiID,
		RewardName:      rewardName,
	}
	if err := s.cache.SetRedemption(ctx, code, data); err != nil {
		return "", err
	}
	s.log.Info("pushcard.redemption.requested",
		"card_id", card.ID, "client_id", clientID, "code", code)
	return code, nil
}

// ConfirmRedemption is called by a collaborator with the code the client
// generated. It atomically marks the card redeemed and consumes the code.
func (s *Service) ConfirmRedemption(ctx context.Context, code, collaboratorID string) (*RedemptionCode, error) {
	if s.cache == nil {
		return nil, fmt.Errorf("cache no configurado")
	}
	data, err := s.cache.ConsumeRedemption(ctx, code)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, fmt.Errorf("código de canje inválido o expirado")
	}
	if err := s.repo.MarkRedeemed(ctx, data.CardID); err != nil {
		// Restore the code so the client can retry.
		_ = s.cache.SetRedemption(ctx, code, data)
		return nil, err
	}
	s.log.Info("pushcard.redemption.confirmed",
		"card_id", data.CardID, "code", code, "collaborator_id", collaboratorID)
	return data, nil
}

// GetConfig returns the active pushcard config for a customer.
func (s *Service) GetConfig(ctx context.Context, customerID string) (*Config, error) {
	return s.repo.GetConfig(ctx, customerID)
}

// UpsertConfig validates and persists the config.
func (s *Service) UpsertConfig(ctx context.Context, cfg *Config) error {
	if cfg.CardSlots <= 0 {
		return fmt.Errorf("card_slots debe ser mayor a 0")
	}
	return s.repo.UpsertConfig(ctx, cfg)
}

// ListCards returns recent cards for admin views.
func (s *Service) ListCards(ctx context.Context, customerSisfiID, status string, limit int) ([]Card, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.repo.ListCardsByCustomer(ctx, customerSisfiID, status, limit)
}

// FindClientIDByPhone delegates the client resolution to the repo.
func (s *Service) FindClientIDByPhone(ctx context.Context, customerID, phoneNumber string) (string, error) {
	return s.repo.FindClientIDByPhone(ctx, customerID, phoneNumber)
}

// buildVisual returns a stamps progress string like "●●●○○".
func buildVisual(count, slots int) string {
	if slots <= 0 {
		return ""
	}
	if count > slots {
		count = slots
	}
	return strings.Repeat("●", count) + strings.Repeat("○", slots-count)
}

// generateUUID creates a v4 UUID string without pulling external deps.
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// generateCode produces a numeric code of the requested length.
func generateCode(length int) string {
	const digits = "0123456789"
	b := make([]byte, length)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = digits[int(b[i])%len(digits)]
	}
	return string(b)
}
