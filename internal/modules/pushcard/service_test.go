package pushcard

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"
)

// fakeRepo is an in-memory implementation of Repository for service tests.
type fakeRepo struct {
	mu      sync.Mutex
	configs map[string]*Config // by customerSisfiID
	cards   map[string]*Card
	stamps  []Stamp
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		configs: make(map[string]*Config),
		cards:   make(map[string]*Card),
	}
}

func (f *fakeRepo) seedConfig(cs string, slots int) *Config {
	f.mu.Lock()
	defer f.mu.Unlock()
	cfg := &Config{CustomerSisfiID: cs, CardSlots: slots, Active: true}
	f.configs[cs] = cfg
	return cfg
}

func (f *fakeRepo) GetConfig(_ context.Context, customerID string) (*Config, error) {
	for _, c := range f.configs {
		if c.CustomerID == customerID && c.Active {
			return c, nil
		}
	}
	return nil, errors.New("not found")
}

func (f *fakeRepo) GetConfigByID(_ context.Context, customerSisfiID string) (*Config, error) {
	c, ok := f.configs[customerSisfiID]
	if !ok {
		return nil, errors.New("not found")
	}
	return c, nil
}

func (f *fakeRepo) UpsertConfig(_ context.Context, cfg *Config) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.configs[cfg.CustomerSisfiID] = cfg
	return nil
}

func (f *fakeRepo) GetOpenCard(_ context.Context, customerSisfiID, clientID string) (*Card, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.cards {
		if c.CustomerSisfiID == customerSisfiID && c.ClientID == clientID && c.Status == StatusOpen {
			return c, nil
		}
	}
	return nil, nil
}

func (f *fakeRepo) GetCard(_ context.Context, cardID string) (*Card, error) {
	c, ok := f.cards[cardID]
	if !ok {
		return nil, errors.New("not found")
	}
	return c, nil
}

func (f *fakeRepo) OpenCard(_ context.Context, customerSisfiID, clientID string) (*Card, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := &Card{
		ID:              generateUUID(),
		CustomerSisfiID: customerSisfiID,
		ClientID:        clientID,
		Status:          StatusOpen,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	f.cards[c.ID] = c
	return c, nil
}

func (f *fakeRepo) CountStamps(_ context.Context, cardID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, s := range f.stamps {
		if s.CardID == cardID {
			n++
		}
	}
	return n, nil
}

func (f *fakeRepo) AddStamp(_ context.Context, stamp *Stamp) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	stamp.CreatedAt = time.Now()
	f.stamps = append(f.stamps, *stamp)
	return nil
}

func (f *fakeRepo) CompleteCard(_ context.Context, cardID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.cards[cardID]
	if !ok || c.Status != StatusOpen {
		return nil
	}
	now := time.Now()
	c.Status = StatusCompleted
	c.CompletedAt = &now
	return nil
}

func (f *fakeRepo) CancelCard(_ context.Context, cardID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.cards[cardID]
	if !ok || c.Status != StatusOpen {
		return nil
	}
	c.Status = StatusCancelled
	return nil
}

func (f *fakeRepo) MarkRedeemed(_ context.Context, cardID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.cards[cardID]
	if !ok || c.Status != StatusCompleted {
		return nil
	}
	c.Status = StatusRedeemed
	return nil
}

func (f *fakeRepo) LastStampByCollaborator(_ context.Context, collaboratorID string, within time.Duration) (*Stamp, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cutoff := time.Now().Add(-within)
	var found *Stamp
	for i := range f.stamps {
		s := f.stamps[i]
		if s.CollaboratorID == collaboratorID && s.CreatedAt.After(cutoff) {
			if found == nil || s.CreatedAt.After(found.CreatedAt) {
				cp := s
				found = &cp
			}
		}
	}
	return found, nil
}

func (f *fakeRepo) DeleteStamp(_ context.Context, stampID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.stamps[:0]
	for _, s := range f.stamps {
		if s.ID != stampID {
			out = append(out, s)
		}
	}
	f.stamps = out
	return nil
}

func (f *fakeRepo) FindClientIDByPhone(_ context.Context, customerID, phone string) (string, error) {
	// Tests don't exercise this path; return empty to keep interface satisfied.
	return "", nil
}

func (f *fakeRepo) ListCardsByCustomer(_ context.Context, customerSisfiID, status string, limit int) ([]Card, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Card
	for _, c := range f.cards {
		if c.CustomerSisfiID != customerSisfiID {
			continue
		}
		if status != "" && c.Status != status {
			continue
		}
		out = append(out, *c)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func newTestService() (*Service, *fakeRepo) {
	repo := newFakeRepo()
	svc := NewService(repo, &fakeCache{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return svc, repo
}

// fakeCache stores redemption codes in memory for service tests.
type fakeCache struct {
	mu sync.Mutex
	m  map[string]*RedemptionCode
}

func (c *fakeCache) ensure() {
	if c.m == nil {
		c.m = make(map[string]*RedemptionCode)
	}
}

func (c *fakeCache) SetRedemption(_ context.Context, code string, data *RedemptionCode) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ensure()
	c.m[code] = data
	return nil
}

func (c *fakeCache) GetRedemption(_ context.Context, code string) (*RedemptionCode, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.m[code], nil
}

func (c *fakeCache) ConsumeRedemption(_ context.Context, code string) (*RedemptionCode, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	d := c.m[code]
	delete(c.m, code)
	return d, nil
}

func (c *fakeCache) DeleteRedemption(_ context.Context, code string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, code)
	return nil
}

func TestAddStamp_OpensCardWhenNoneExists(t *testing.T) {
	svc, repo := newTestService()
	repo.seedConfig("cs-1", 5)

	res, err := svc.AddStamp(context.Background(), AddStampReq{
		CustomerSisfiID: "cs-1", ClientID: "client-1", CollaboratorID: "collab-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StampsCount != 1 {
		t.Fatalf("want 1 stamp, got %d", res.StampsCount)
	}
	if res.Completed {
		t.Fatalf("should not be completed yet")
	}
	if res.Card.Status != StatusOpen {
		t.Fatalf("expected open card, got %s", res.Card.Status)
	}
}

func TestAddStamp_IncrementsAcrossCalls(t *testing.T) {
	svc, repo := newTestService()
	repo.seedConfig("cs-1", 5)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		res, err := svc.AddStamp(ctx, AddStampReq{
			CustomerSisfiID: "cs-1", ClientID: "c", CollaboratorID: "k",
		})
		if err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}
		if res.StampsCount != i {
			t.Fatalf("iter %d: want %d stamps, got %d", i, i, res.StampsCount)
		}
	}
}

func TestAddStamp_CompletesAtSlots(t *testing.T) {
	svc, repo := newTestService()
	repo.seedConfig("cs-1", 3)
	ctx := context.Background()

	var last *AddStampResult
	for i := 0; i < 3; i++ {
		r, err := svc.AddStamp(ctx, AddStampReq{
			CustomerSisfiID: "cs-1", ClientID: "c", CollaboratorID: "k",
		})
		if err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}
		last = r
	}
	if !last.Completed {
		t.Fatalf("expected completed=true on last stamp")
	}
	if last.Card.Status != StatusCompleted {
		t.Fatalf("expected status=completed, got %s", last.Card.Status)
	}
}

func TestAddStamp_RejectsMissingFields(t *testing.T) {
	svc, _ := newTestService()
	if _, err := svc.AddStamp(context.Background(), AddStampReq{ClientID: "c"}); err == nil {
		t.Fatalf("expected error for missing fields")
	}
}

func TestUndoLastStamp_RemovesWithinWindow(t *testing.T) {
	svc, repo := newTestService()
	repo.seedConfig("cs-1", 5)
	ctx := context.Background()

	_, _ = svc.AddStamp(ctx, AddStampReq{CustomerSisfiID: "cs-1", ClientID: "c", CollaboratorID: "k"})
	if _, err := svc.UndoLastStamp(ctx, "k"); err != nil {
		t.Fatalf("undo: %v", err)
	}
	prog, err := svc.GetProgress(ctx, "cs-1", "c")
	if err != nil {
		t.Fatal(err)
	}
	if prog.StampsCount != 0 {
		t.Fatalf("want 0 stamps after undo, got %d", prog.StampsCount)
	}
}

func TestUndoLastStamp_NoneToUndo(t *testing.T) {
	svc, _ := newTestService()
	if _, err := svc.UndoLastStamp(context.Background(), "k"); !errors.Is(err, ErrNoStampToUndo) {
		t.Fatalf("expected ErrNoStampToUndo, got %v", err)
	}
}

// TestUndoLastStamp_RejectsUndoOnExpiredCancelledCard (LG-3) covers the
// undo+expiración interaction: a card with expiry configured accrues a stamp,
// then is auto-cancelled because it went stale, then the collaborator tries to
// undo within the correction window. The stamp still exists (append-only) and
// LastStampByCollaborator will find it, but its card is now cancelled. Undoing
// would silently DeleteStamp from a terminal-state card and drop the stamp with
// no useful effect, so the service must reject the undo (ErrCardCancelled) and
// leave the stamp intact rather than losing it silently.
func TestUndoLastStamp_RejectsUndoOnExpiredCancelledCard(t *testing.T) {
	svc, repo := newTestService()
	cfg := repo.seedConfig("cs-1", 5)
	cfg.CardExpiryDays = intPtr(7)
	ctx := context.Background()

	// A stamp opens a card.
	r1, err := svc.AddStamp(ctx, AddStampReq{CustomerSisfiID: "cs-1", ClientID: "c", CollaboratorID: "k"})
	if err != nil {
		t.Fatal(err)
	}
	cardID := r1.Card.ID

	// Age the card past expiry, then trigger lazy expiration via GetProgress.
	// The stamp's created_at stays recent, so it remains within CorrectionWindow.
	repo.backdateOpenCard("cs-1", "c", 8*24*time.Hour)
	if _, err := svc.GetProgress(ctx, "cs-1", "c"); err != nil {
		t.Fatal(err)
	}
	if old, _ := repo.GetCard(ctx, cardID); old.Status != StatusCancelled {
		t.Fatalf("precondition: expected card cancelled, got %s", old.Status)
	}

	// Undo must be rejected rather than silently deleting the stamp.
	if _, err := svc.UndoLastStamp(ctx, "k"); !errors.Is(err, ErrCardCancelled) {
		t.Fatalf("expected ErrCardCancelled, got %v", err)
	}

	// The stamp must NOT have been lost: it is still present on the cancelled card.
	n, err := repo.CountStamps(ctx, cardID)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("stamp was silently lost: want 1 stamp on cancelled card, got %d", n)
	}
}

func TestGetProgress_NoOpenCard(t *testing.T) {
	svc, repo := newTestService()
	repo.seedConfig("cs-1", 5)
	prog, err := svc.GetProgress(context.Background(), "cs-1", "c")
	if err != nil {
		t.Fatal(err)
	}
	if prog.HasOpenCard {
		t.Fatalf("expected no open card")
	}
	if prog.Visual != "○○○○○" {
		t.Fatalf("expected empty visual, got %q", prog.Visual)
	}
}

func TestUpsertConfig_RejectsZeroSlots(t *testing.T) {
	svc, _ := newTestService()
	err := svc.UpsertConfig(context.Background(), &Config{CustomerSisfiID: "cs-1", CardSlots: 0})
	if err == nil {
		t.Fatalf("expected error for slots=0")
	}
}

func TestBuildVisual(t *testing.T) {
	cases := []struct {
		count, slots int
		want         string
	}{
		{0, 5, "○○○○○"},
		{3, 5, "●●●○○"},
		{5, 5, "●●●●●"},
		{6, 5, "●●●●●"},
		{0, 0, ""},
	}
	for _, c := range cases {
		got := buildVisual(c.count, c.slots)
		if got != c.want {
			t.Errorf("buildVisual(%d,%d) = %q, want %q", c.count, c.slots, got, c.want)
		}
	}
}

func TestRequestRedemption_RequiresCompletedCard(t *testing.T) {
	svc, repo := newTestService()
	repo.seedConfig("cs-1", 3)
	ctx := context.Background()

	// No completed card → error
	if _, err := svc.RequestRedemption(ctx, "cs-1", "client-1", "cust-1", "Café gratis"); err == nil {
		t.Fatalf("expected error when no completed card")
	}
}

func TestRequestRedemption_WithCompletedCardReturnsCode(t *testing.T) {
	svc, repo := newTestService()
	repo.seedConfig("cs-1", 2)
	ctx := context.Background()

	// Complete a card
	for i := 0; i < 2; i++ {
		if _, err := svc.AddStamp(ctx, AddStampReq{CustomerSisfiID: "cs-1", ClientID: "c", CollaboratorID: "k"}); err != nil {
			t.Fatal(err)
		}
	}
	code, err := svc.RequestRedemption(ctx, "cs-1", "c", "cust-1", "Café")
	if err != nil {
		t.Fatalf("request redemption: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("want 6-char code, got %d (%q)", len(code), code)
	}
}

// backdateOpenCard shifts the created_at of the client's open card into the
// past so expiry logic can be exercised deterministically.
func (f *fakeRepo) backdateOpenCard(customerSisfiID, clientID string, age time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.cards {
		if c.CustomerSisfiID == customerSisfiID && c.ClientID == clientID && c.Status == StatusOpen {
			c.CreatedAt = time.Now().Add(-age)
		}
	}
}

func intPtr(n int) *int { return &n }

func TestUpsertConfig_RejectsZeroExpiry(t *testing.T) {
	svc, _ := newTestService()
	err := svc.UpsertConfig(context.Background(), &Config{
		CustomerSisfiID: "cs-1", CardSlots: 5, CardExpiryDays: intPtr(0),
	})
	if err == nil {
		t.Fatalf("expected error for card_expiry_days=0")
	}
}

func TestUpsertConfig_AcceptsExpiry(t *testing.T) {
	svc, repo := newTestService()
	err := svc.UpsertConfig(context.Background(), &Config{
		CustomerSisfiID: "cs-1", CardSlots: 5, CardExpiryDays: intPtr(30),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := repo.GetConfigByID(context.Background(), "cs-1")
	if got.CardExpiryDays == nil || *got.CardExpiryDays != 30 {
		t.Fatalf("expected persisted expiry=30, got %v", got.CardExpiryDays)
	}
}

func TestAddStamp_ExpiresStaleCardAndOpensFresh(t *testing.T) {
	svc, repo := newTestService()
	cfg := repo.seedConfig("cs-1", 5)
	cfg.CardExpiryDays = intPtr(7)
	ctx := context.Background()

	// First stamp opens a card.
	r1, err := svc.AddStamp(ctx, AddStampReq{CustomerSisfiID: "cs-1", ClientID: "c", CollaboratorID: "k"})
	if err != nil {
		t.Fatal(err)
	}
	firstCardID := r1.Card.ID

	// Age the card past its expiry window.
	repo.backdateOpenCard("cs-1", "c", 8*24*time.Hour)

	// GetProgress should now report no open card (the stale one was cancelled).
	prog, err := svc.GetProgress(ctx, "cs-1", "c")
	if err != nil {
		t.Fatal(err)
	}
	if prog.HasOpenCard {
		t.Fatalf("expected stale card to be expired, still open")
	}
	if old, _ := repo.GetCard(ctx, firstCardID); old.Status != StatusCancelled {
		t.Fatalf("expected first card cancelled, got %s", old.Status)
	}

	// Next stamp opens a fresh card with count reset to 1.
	r2, err := svc.AddStamp(ctx, AddStampReq{CustomerSisfiID: "cs-1", ClientID: "c", CollaboratorID: "k"})
	if err != nil {
		t.Fatal(err)
	}
	if r2.Card.ID == firstCardID {
		t.Fatalf("expected a new card, reused expired one")
	}
	if r2.StampsCount != 1 {
		t.Fatalf("expected fresh card with 1 stamp, got %d", r2.StampsCount)
	}
}

func TestAddStamp_NoExpiryKeepsCard(t *testing.T) {
	svc, repo := newTestService()
	repo.seedConfig("cs-1", 5) // CardExpiryDays nil = no expiration (default)
	ctx := context.Background()

	r1, err := svc.AddStamp(ctx, AddStampReq{CustomerSisfiID: "cs-1", ClientID: "c", CollaboratorID: "k"})
	if err != nil {
		t.Fatal(err)
	}
	firstCardID := r1.Card.ID

	// Even a very old card must survive when expiry is disabled.
	repo.backdateOpenCard("cs-1", "c", 365*24*time.Hour)

	r2, err := svc.AddStamp(ctx, AddStampReq{CustomerSisfiID: "cs-1", ClientID: "c", CollaboratorID: "k"})
	if err != nil {
		t.Fatal(err)
	}
	if r2.Card.ID != firstCardID {
		t.Fatalf("expected same card when expiry disabled")
	}
	if r2.StampsCount != 2 {
		t.Fatalf("expected 2 stamps on same card, got %d", r2.StampsCount)
	}
}

func TestConfirmRedemption_MarksRedeemed(t *testing.T) {
	svc, repo := newTestService()
	repo.seedConfig("cs-1", 2)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		_, _ = svc.AddStamp(ctx, AddStampReq{CustomerSisfiID: "cs-1", ClientID: "c", CollaboratorID: "k"})
	}
	code, _ := svc.RequestRedemption(ctx, "cs-1", "c", "cust-1", "Café")

	data, err := svc.ConfirmRedemption(ctx, code, "k")
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if data.RewardName != "Café" {
		t.Fatalf("want Café, got %s", data.RewardName)
	}

	// Card should be redeemed now
	card, _ := repo.GetCard(ctx, data.CardID)
	if card.Status != StatusRedeemed {
		t.Fatalf("want redeemed, got %s", card.Status)
	}

	// Code should not be reusable
	if _, err := svc.ConfirmRedemption(ctx, code, "k"); err == nil {
		t.Fatalf("code should not be reusable")
	}
}
