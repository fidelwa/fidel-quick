package resolver

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoleResolve_Collaborator(t *testing.T) {
	repo := &mockRepo{
		findCollaboratorFn: func(_ context.Context, _, _ string) (string, error) {
			return "collab-123", nil
		},
	}
	resolver := NewRoleResolver(repo)

	result, err := resolver.Resolve(context.Background(), "+123", "cust-1")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "collaborator", result.Role)
	assert.Equal(t, "collab-123", result.UserID)
}

func TestRoleResolve_Client(t *testing.T) {
	repo := &mockRepo{
		findCollaboratorFn: func(_ context.Context, _, _ string) (string, error) {
			return "", nil // not a collaborator
		},
		findClientFn: func(_ context.Context, _, _ string) (string, error) {
			return "client-456", nil
		},
	}
	resolver := NewRoleResolver(repo)

	result, err := resolver.Resolve(context.Background(), "+123", "cust-1")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "client", result.Role)
	assert.Equal(t, "client-456", result.UserID)
}

func TestRoleResolve_CollaboratorPriority(t *testing.T) {
	// When user is both collaborator and client, collaborator wins
	repo := &mockRepo{
		findCollaboratorFn: func(_ context.Context, _, _ string) (string, error) {
			return "collab-123", nil
		},
		findClientFn: func(_ context.Context, _, _ string) (string, error) {
			return "client-456", nil // should not be reached
		},
	}
	resolver := NewRoleResolver(repo)

	result, err := resolver.Resolve(context.Background(), "+123", "cust-1")

	require.NoError(t, err)
	assert.Equal(t, "collaborator", result.Role)
}

func TestRoleResolve_NotFound(t *testing.T) {
	repo := &mockRepo{
		findCollaboratorFn: func(_ context.Context, _, _ string) (string, error) {
			return "", nil
		},
		findClientFn: func(_ context.Context, _, _ string) (string, error) {
			return "", nil
		},
	}
	resolver := NewRoleResolver(repo)

	result, err := resolver.Resolve(context.Background(), "+123", "cust-1")

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestRoleResolve_CollaboratorError(t *testing.T) {
	repo := &mockRepo{
		findCollaboratorFn: func(_ context.Context, _, _ string) (string, error) {
			return "", fmt.Errorf("db error")
		},
	}
	resolver := NewRoleResolver(repo)

	_, err := resolver.Resolve(context.Background(), "+123", "cust-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "find collaborator")
}

func TestRoleResolve_ClientError(t *testing.T) {
	repo := &mockRepo{
		findCollaboratorFn: func(_ context.Context, _, _ string) (string, error) {
			return "", nil
		},
		findClientFn: func(_ context.Context, _, _ string) (string, error) {
			return "", fmt.Errorf("db error")
		},
	}
	resolver := NewRoleResolver(repo)

	_, err := resolver.Resolve(context.Background(), "+123", "cust-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "find client")
}
