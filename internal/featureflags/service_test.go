package featureflags

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/theluisbolivar/fidel-quick/internal/apperror"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// --- Resolver precedence: override > global > default ---

func TestFlag_Resolve(t *testing.T) {
	const cust = "cust-1"
	other := "cust-2"

	tests := []struct {
		name       string
		flag       Flag
		customerID string
		want       bool
	}{
		{
			name:       "override true beats global false and default false",
			flag:       Flag{CustomerOverrides: map[string]bool{cust: true}, EnabledGlobally: false, DefaultValue: false},
			customerID: cust,
			want:       true,
		},
		{
			name:       "override false beats global true",
			flag:       Flag{CustomerOverrides: map[string]bool{cust: false}, EnabledGlobally: true, DefaultValue: true},
			customerID: cust,
			want:       false,
		},
		{
			name:       "no override for this customer falls through to global true",
			flag:       Flag{CustomerOverrides: map[string]bool{other: false}, EnabledGlobally: true, DefaultValue: false},
			customerID: cust,
			want:       true,
		},
		{
			name:       "global false falls through to default true",
			flag:       Flag{EnabledGlobally: false, DefaultValue: true},
			customerID: cust,
			want:       true,
		},
		{
			name:       "global false and default false is false",
			flag:       Flag{EnabledGlobally: false, DefaultValue: false},
			customerID: cust,
			want:       false,
		},
		{
			name:       "empty customerID skips override, uses global",
			flag:       Flag{CustomerOverrides: map[string]bool{cust: false}, EnabledGlobally: true},
			customerID: "",
			want:       true,
		},
		{
			name:       "nil overrides map does not panic, uses global",
			flag:       Flag{CustomerOverrides: nil, EnabledGlobally: true},
			customerID: cust,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.flag.Resolve(tt.customerID); got != tt.want {
				t.Fatalf("Resolve(%q) = %v, want %v", tt.customerID, got, tt.want)
			}
		})
	}
}

// --- Fakes ---

type fakeRepo struct {
	flags     map[string]*Flag
	getErr    error
	listErr   error
	upsertErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{flags: map[string]*Flag{}}
}

func (r *fakeRepo) List(_ context.Context) ([]Flag, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	out := make([]Flag, 0, len(r.flags))
	for _, f := range r.flags {
		out = append(out, *f)
	}
	return out, nil
}

func (r *fakeRepo) Get(_ context.Context, key string) (*Flag, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	f, ok := r.flags[key]
	if !ok {
		return nil, apperror.NotFound("feature flag not found", nil)
	}
	cp := *f
	return &cp, nil
}

func (r *fakeRepo) Upsert(_ context.Context, key string, in UpdateInput) (*Flag, error) {
	if r.upsertErr != nil {
		return nil, r.upsertErr
	}
	f, ok := r.flags[key]
	if !ok {
		f = &Flag{Key: key, CustomerOverrides: map[string]bool{}}
	}
	if in.EnabledGlobally != nil {
		f.EnabledGlobally = *in.EnabledGlobally
	}
	if in.DefaultValue != nil {
		f.DefaultValue = *in.DefaultValue
	}
	if in.Description != nil {
		f.Description = *in.Description
	}
	if in.CustomerOverrides != nil {
		f.CustomerOverrides = in.CustomerOverrides
	}
	r.flags[key] = f
	cp := *f
	return &cp, nil
}

// countingCache records deletes so we can assert cache invalidation.
type countingCache struct {
	store   map[string]*Flag
	deletes int
	sets    int
	hits    int
}

func newCountingCache() *countingCache {
	return &countingCache{store: map[string]*Flag{}}
}

func (c *countingCache) Get(_ context.Context, key string) (*Flag, bool) {
	f, ok := c.store[key]
	if ok {
		c.hits++
		cp := *f
		return &cp, true
	}
	return nil, false
}

func (c *countingCache) Set(_ context.Context, key string, flag *Flag) {
	c.sets++
	cp := *flag
	c.store[key] = &cp
}

func (c *countingCache) Delete(_ context.Context, key string) {
	c.deletes++
	delete(c.store, key)
}

// --- Service ---

func TestService_Enabled_UnknownFlagIsFalse(t *testing.T) {
	svc := NewService(newFakeRepo(), newCountingCache(), discardLogger())
	if svc.Enabled(context.Background(), "does.not.exist", "cust-1") {
		t.Fatal("unknown flag should resolve to false")
	}
}

func TestService_Enabled_ResolvesPrecedence(t *testing.T) {
	repo := newFakeRepo()
	repo.flags["f"] = &Flag{Key: "f", CustomerOverrides: map[string]bool{"c1": true}, EnabledGlobally: false}
	svc := NewService(repo, newCountingCache(), discardLogger())

	if !svc.Enabled(context.Background(), "f", "c1") {
		t.Fatal("override true should win")
	}
	if svc.Enabled(context.Background(), "f", "c2") {
		t.Fatal("no override + global false should be false")
	}
}

func TestService_Enabled_UsesAndPopulatesCache(t *testing.T) {
	repo := newFakeRepo()
	repo.flags["f"] = &Flag{Key: "f", EnabledGlobally: true}
	cache := newCountingCache()
	svc := NewService(repo, cache, discardLogger())

	_ = svc.Enabled(context.Background(), "f", "c1") // miss → repo → set
	if cache.sets != 1 {
		t.Fatalf("expected 1 cache set, got %d", cache.sets)
	}
	_ = svc.Enabled(context.Background(), "f", "c1") // hit
	if cache.hits != 1 {
		t.Fatalf("expected 1 cache hit, got %d", cache.hits)
	}
}

func TestService_Enabled_RepoErrorIsFalse(t *testing.T) {
	repo := newFakeRepo()
	repo.getErr = apperror.Internal("db down", nil)
	svc := NewService(repo, newCountingCache(), discardLogger())
	if svc.Enabled(context.Background(), "f", "c1") {
		t.Fatal("repo error should resolve to false, not panic")
	}
}

func TestService_Update_InvalidatesCache(t *testing.T) {
	repo := newFakeRepo()
	repo.flags["f"] = &Flag{Key: "f", EnabledGlobally: false}
	cache := newCountingCache()
	svc := NewService(repo, cache, discardLogger())

	// Prime cache.
	_ = svc.Enabled(context.Background(), "f", "c1")
	if cache.sets != 1 {
		t.Fatalf("expected cache primed, sets=%d", cache.sets)
	}

	on := true
	if _, err := svc.Update(context.Background(), "f", UpdateInput{EnabledGlobally: &on}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if cache.deletes != 1 {
		t.Fatalf("expected 1 cache delete on update, got %d", cache.deletes)
	}
	// Next read reflects the new value.
	if !svc.Enabled(context.Background(), "f", "c1") {
		t.Fatal("after enabling globally, flag should be true")
	}
}

func TestService_EnabledFor_ResolvesAllFlags(t *testing.T) {
	repo := newFakeRepo()
	repo.flags["a"] = &Flag{Key: "a", EnabledGlobally: true}
	repo.flags["b"] = &Flag{Key: "b", CustomerOverrides: map[string]bool{"c1": true}}
	repo.flags["c"] = &Flag{Key: "c", DefaultValue: false}
	svc := NewService(repo, newCountingCache(), discardLogger())

	got, err := svc.EnabledFor(context.Background(), "c1")
	if err != nil {
		t.Fatalf("EnabledFor: %v", err)
	}
	want := map[string]bool{"a": true, "b": true, "c": false}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("flag %q = %v, want %v", k, got[k], v)
		}
	}
}
