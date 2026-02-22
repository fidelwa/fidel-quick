package resolver

import (
	"context"
	"fmt"
)

// RoleResult holds the resolved role for a user in a specific business.
type RoleResult struct {
	Role   string // "client" or "collaborator"
	UserID string // the client_id or collaborator_id
}

type RoleResolver struct {
	repo Repository
}

func NewRoleResolver(repo Repository) *RoleResolver {
	return &RoleResolver{repo: repo}
}

// Resolve determines the user's role in a given business.
// Collaborator takes priority if the user is both.
func (r *RoleResolver) Resolve(ctx context.Context, phone, customerID string) (*RoleResult, error) {
	// Check collaborator first (priority)
	collabID, err := r.repo.FindCollaborator(ctx, phone, customerID)
	if err != nil {
		return nil, fmt.Errorf("find collaborator: %w", err)
	}
	if collabID != "" {
		return &RoleResult{Role: "collaborator", UserID: collabID}, nil
	}

	// Check client
	clientID, err := r.repo.FindClient(ctx, phone, customerID)
	if err != nil {
		return nil, fmt.Errorf("find client: %w", err)
	}
	if clientID != "" {
		return &RoleResult{Role: "client", UserID: clientID}, nil
	}

	return nil, nil
}
