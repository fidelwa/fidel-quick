package resolver

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theluisbolivar/fidel-quick/internal/session"
)

// --- Mock Repository ---

type mockRepo struct {
	getActiveCustomerByIDFn  func(ctx context.Context, id string) (string, error)
	userExistsInBusinessFn   func(ctx context.Context, phone, customerID string) (bool, error)
	findBusinessesByPhoneFn  func(ctx context.Context, phone string) ([]session.SelectionOption, error)
	findCollaboratorFn       func(ctx context.Context, phone, customerID string) (string, error)
	findClientFn             func(ctx context.Context, phone, customerID string) (string, error)
	registerClientFn         func(ctx context.Context, customerID, phone string) error
	getCustomerBySlugFn      func(ctx context.Context, slug string) (string, string, string, string, string, string, error)
	getActiveProgramTypesFn  func(ctx context.Context, customerID string) ([]string, error)
}

func (m *mockRepo) GetActiveCustomerByID(ctx context.Context, id string) (string, error) {
	if m.getActiveCustomerByIDFn != nil {
		return m.getActiveCustomerByIDFn(ctx, id)
	}
	return "", nil
}
func (m *mockRepo) UserExistsInBusiness(ctx context.Context, phone, customerID string) (bool, error) {
	if m.userExistsInBusinessFn != nil {
		return m.userExistsInBusinessFn(ctx, phone, customerID)
	}
	return false, nil
}
func (m *mockRepo) FindBusinessesByPhone(ctx context.Context, phone string) ([]session.SelectionOption, error) {
	if m.findBusinessesByPhoneFn != nil {
		return m.findBusinessesByPhoneFn(ctx, phone)
	}
	return nil, nil
}
func (m *mockRepo) FindCollaborator(ctx context.Context, phone, customerID string) (string, error) {
	if m.findCollaboratorFn != nil {
		return m.findCollaboratorFn(ctx, phone, customerID)
	}
	return "", nil
}
func (m *mockRepo) FindClient(ctx context.Context, phone, customerID string) (string, error) {
	if m.findClientFn != nil {
		return m.findClientFn(ctx, phone, customerID)
	}
	return "", nil
}
func (m *mockRepo) RegisterClient(ctx context.Context, customerID, phone string) error {
	if m.registerClientFn != nil {
		return m.registerClientFn(ctx, customerID, phone)
	}
	return nil
}
func (m *mockRepo) GetCustomerBySlug(ctx context.Context, slug string) (string, string, string, string, string, string, error) {
	if m.getCustomerBySlugFn != nil {
		return m.getCustomerBySlugFn(ctx, slug)
	}
	return "", "", "", "", "", "", fmt.Errorf("not found")
}
func (m *mockRepo) GetActiveProgramTypes(ctx context.Context, customerID string) ([]string, error) {
	if m.getActiveProgramTypesFn != nil {
		return m.getActiveProgramTypesFn(ctx, customerID)
	}
	return nil, nil
}

// --- Business Resolver Tests ---

func TestResolve_Deeplink_ExistingUser(t *testing.T) {
	repo := &mockRepo{
		getActiveCustomerByIDFn: func(_ context.Context, id string) (string, error) {
			return "Test Business", nil
		},
		userExistsInBusinessFn: func(_ context.Context, _, _ string) (bool, error) {
			return true, nil
		},
	}
	resolver := NewBusinessResolver(repo)

	result, multi, err := resolver.Resolve(context.Background(), "+123", "unirme:cust-uuid-123")

	require.NoError(t, err)
	assert.Nil(t, multi)
	assert.NotNil(t, result)
	assert.Equal(t, "cust-uuid-123", result.CustomerID)
	assert.Equal(t, "Test Business", result.BusinessName)
	assert.False(t, result.IsNew)
}

func TestResolve_Deeplink_NewUser(t *testing.T) {
	repo := &mockRepo{
		getActiveCustomerByIDFn: func(_ context.Context, _ string) (string, error) {
			return "Test Business", nil
		},
		userExistsInBusinessFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, nil
		},
	}
	resolver := NewBusinessResolver(repo)

	result, multi, err := resolver.Resolve(context.Background(), "+123", "unirme:cust-uuid-123")

	require.NoError(t, err)
	assert.Nil(t, multi)
	assert.NotNil(t, result)
	assert.True(t, result.IsNew)
}

func TestResolve_Deeplink_BusinessNotFound(t *testing.T) {
	repo := &mockRepo{
		getActiveCustomerByIDFn: func(_ context.Context, _ string) (string, error) {
			return "", nil // empty name = not found
		},
	}
	resolver := NewBusinessResolver(repo)

	result, multiResult, err := resolver.Resolve(context.Background(), "+123", "unirme:bad-uuid")

	require.NoError(t, err)
	assert.Nil(t, result)
	assert.Nil(t, multiResult)
}

func TestResolve_Phone_SingleBusiness(t *testing.T) {
	repo := &mockRepo{
		findBusinessesByPhoneFn: func(_ context.Context, _ string) ([]session.SelectionOption, error) {
			return []session.SelectionOption{
				{CustomerID: "cust-1", Name: "Business A"},
			}, nil
		},
	}
	resolver := NewBusinessResolver(repo)

	result, multi, err := resolver.Resolve(context.Background(), "+123", "Hola")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, multi)
	assert.Equal(t, "cust-1", result.CustomerID)
	assert.Equal(t, "Business A", result.BusinessName)
}

func TestResolve_Phone_MultipleBusinesses(t *testing.T) {
	repo := &mockRepo{
		findBusinessesByPhoneFn: func(_ context.Context, _ string) ([]session.SelectionOption, error) {
			return []session.SelectionOption{
				{CustomerID: "cust-1", Name: "Business A"},
				{CustomerID: "cust-2", Name: "Business B"},
			}, nil
		},
	}
	resolver := NewBusinessResolver(repo)

	result, multi, err := resolver.Resolve(context.Background(), "+123", "Hola")

	require.NoError(t, err)
	assert.Nil(t, result)
	assert.NotNil(t, multi)
	assert.Len(t, multi.Options, 2)
}

func TestResolve_Phone_NotRegistered(t *testing.T) {
	repo := &mockRepo{
		findBusinessesByPhoneFn: func(_ context.Context, _ string) ([]session.SelectionOption, error) {
			return nil, nil
		},
	}
	resolver := NewBusinessResolver(repo)

	result, multi, err := resolver.Resolve(context.Background(), "+123", "Hola")

	require.NoError(t, err)
	assert.Nil(t, result)
	assert.Nil(t, multi)
}

func TestExtractDeeplink(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		found    bool
	}{
		{"unirme:abc-123", "abc-123", true},
		{"Hola, quiero unirme:abc-123", "abc-123", true},
		{"unirme:abc-123 extra text", "abc-123", true},
		{"hola mundo", "", false},
		{"unirme:", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		id, ok := extractDeeplink(tt.input)
		assert.Equal(t, tt.found, ok, "input: %q", tt.input)
		if tt.found {
			assert.Equal(t, tt.expected, id, "input: %q", tt.input)
		}
	}
}
