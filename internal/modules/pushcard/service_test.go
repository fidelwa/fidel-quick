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
	svc := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return svc, repo
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
