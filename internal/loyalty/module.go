package loyalty

import (
	"context"

	"github.com/gin-gonic/gin"
)

// Module is the contract every loyalty system must implement.
// To add a new loyalty type (cashback, tiers, etc.):
// 1. Create internal/modules/{name}/
// 2. Implement this interface
// 3. Register in main.go
type Module interface {
	// Name returns the module identifier (e.g., "earn_burn").
	Name() string

	// Menus returns interactive WhatsApp menu definitions per role.
	Menus() map[string][]MenuDefinition // role → menu options

	// Prefixes returns selection ID prefixes this module handles (e.g., ["reward:"]).
	Prefixes() []string

	// SelectionFlow returns the flow command ID and data key for a given prefix.
	// e.g., ("reward:", ) → ("request_redemption", "reward_id")
	SelectionFlow(prefix string) (commandID string, dataKey string)

	// HandleCommand processes a menu selection or completed flow.
	HandleCommand(ctx context.Context, cmd Command) (*CommandResult, error)

	// FlowDefinitions returns step-by-step flows for each command.
	FlowDefinitions() map[string]FlowDefinition // command_id → flow

	// RegisterRoutes adds this module's REST API routes.
	RegisterRoutes(rg *gin.RouterGroup)
}
