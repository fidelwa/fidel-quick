package loyalty

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
)

// Registry manages all loyalty modules and dispatches commands.
type Registry struct {
	modules  map[string]Module
	commands map[string]string // command_id → module_name
}

func NewRegistry() *Registry {
	return &Registry{
		modules:  make(map[string]Module),
		commands: make(map[string]string),
	}
}

// Register adds a module to the registry and indexes its commands.
func (r *Registry) Register(m Module) {
	r.modules[m.Name()] = m

	// Index all menu commands to this module
	for _, menus := range m.Menus() {
		for _, menu := range menus {
			r.commands[menu.ID] = m.Name()
		}
	}

	// Index all flow commands
	for cmdID := range m.FlowDefinitions() {
		r.commands[cmdID] = m.Name()
	}
}

// AllMenus returns aggregated menu options for a given role across all modules.
func (r *Registry) AllMenus(role string) []MenuDefinition {
	var menus []MenuDefinition
	for _, m := range r.modules {
		if roleMenus, ok := m.Menus()[role]; ok {
			menus = append(menus, roleMenus...)
		}
	}
	return menus
}

// FilteredMenus returns menu options for a role, filtered to only include modules in activeModules.
// If activeModules is empty, returns all menus (fallback for stale sessions).
func (r *Registry) FilteredMenus(role string, activeModules []string) []MenuDefinition {
	if len(activeModules) == 0 {
		return r.AllMenus(role)
	}
	var menus []MenuDefinition
	for _, m := range r.modules {
		if !containsStr(activeModules, m.Name()) {
			continue
		}
		if roleMenus, ok := m.Menus()[role]; ok {
			menus = append(menus, roleMenus...)
		}
	}
	return menus
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// GetFlowDefinition returns the flow definition for a command, if any.
func (r *Registry) GetFlowDefinition(commandID string) (*FlowDefinition, bool) {
	moduleName, ok := r.commands[commandID]
	if !ok {
		return nil, false
	}
	m := r.modules[moduleName]
	flows := m.FlowDefinitions()
	if flow, exists := flows[commandID]; exists {
		return &flow, true
	}
	return nil, false
}

// Dispatch routes a command to the correct module's HandleCommand.
func (r *Registry) Dispatch(ctx context.Context, cmd Command) (*CommandResult, error) {
	moduleName, ok := r.commands[cmd.ID]
	if !ok {
		return nil, fmt.Errorf("unknown command: %s", cmd.ID)
	}
	m := r.modules[moduleName]
	return m.HandleCommand(ctx, cmd)
}

// RegisterAllRoutes registers all module REST API routes.
func (r *Registry) RegisterAllRoutes(rg *gin.RouterGroup) {
	for _, m := range r.modules {
		m.RegisterRoutes(rg)
	}
}
