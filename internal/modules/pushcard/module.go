package pushcard

import (
	"context"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/theluisbolivar/fidel-quick/internal/loyalty"
)

// Module wires pushcard into the loyalty.Registry.
type Module struct {
	service *Service
	api     *APIHandler
}

func NewModule(service *Service, api *APIHandler) *Module {
	return &Module{service: service, api: api}
}

func (m *Module) Name() string { return "pushcard" }

func (m *Module) Menus() map[string][]loyalty.MenuDefinition {
	return map[string][]loyalty.MenuDefinition{
		"client":       ClientMenus(),
		"collaborator": CollaboratorMenus(),
	}
}

func (m *Module) FlowDefinitions() map[string]loyalty.FlowDefinition {
	return FlowDefs()
}

// Prefixes is empty for pushcard — there are no per-card option lists in the
// MVP. (Cashback uses cb_reward: for picking a reward from a list; pushcard's
// reward is fixed in config.) Returning an empty slice keeps the engine happy.
func (m *Module) Prefixes() []string { return []string{} }

func (m *Module) SelectionFlow(prefix string) (string, string) {
	return "", ""
}

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	if m.api != nil {
		m.api.RegisterRoutes(rg)
	}
}

func (m *Module) HandleCommand(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	switch cmd.ID {
	case "pc_check_card":
		return m.handleCheckCard(ctx, cmd)
	case "pc_redeem":
		return m.handleRedeem(ctx, cmd)
	case "pc_add_stamp":
		return m.handleAddStamp(ctx, cmd)
	case "pc_undo_stamp":
		return m.handleUndoStamp(ctx, cmd)
	case "pc_confirm_redemption":
		return m.handleConfirmRedemption(ctx, cmd)
	default:
		return nil, fmt.Errorf("unknown pushcard command: %s", cmd.ID)
	}
}

// handleCheckCard returns the client's current pushcard progress.
func (m *Module) handleCheckCard(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	cfg, err := m.service.GetConfig(ctx, cmd.UserContext.CustomerID)
	if err != nil {
		return nil, err
	}

	progress, err := m.service.GetProgress(ctx, cfg.CustomerSisfiID, cmd.UserContext.UserID)
	if err != nil {
		return nil, err
	}

	if !progress.HasOpenCard {
		msg := fmt.Sprintf("Aún no tenés sellos en tu tarjeta de *%s*.\nPedí tu primer sello al colaborador.", cfg.Name)
		return &loyalty.CommandResult{Message: msg}, nil
	}

	msg := fmt.Sprintf("*%s* — tu tarjeta\n%s\n%d / %d sellos",
		cfg.Name, progress.Visual, progress.StampsCount, progress.CardSlots)
	return &loyalty.CommandResult{Message: msg}, nil
}

// handleRedeem is a placeholder until the full redemption flow lands in FID-4.
func (m *Module) handleRedeem(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	cfg, err := m.service.GetConfig(ctx, cmd.UserContext.CustomerID)
	if err != nil {
		return nil, err
	}
	progress, err := m.service.GetProgress(ctx, cfg.CustomerSisfiID, cmd.UserContext.UserID)
	if err != nil {
		return nil, err
	}
	if progress.HasOpenCard && progress.StampsCount < progress.CardSlots {
		msg := fmt.Sprintf("Aún no completás la tarjeta. Te faltan %d sellos.", progress.CardSlots-progress.StampsCount)
		return &loyalty.CommandResult{Message: msg}, nil
	}
	return &loyalty.CommandResult{
		Message: "Mostrale al colaborador esta pantalla. (Flujo completo de canje llega en FID-4.)",
	}, nil
}

// handleAddStamp is a stub that uses the collaborator + provided client phone.
// The full collaborator flow lives in FID-5; here we accept the data already
// collected by FlowDefs and let the service do the work.
func (m *Module) handleAddStamp(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	clientID := cmd.Data["client_id"]
	if clientID == "" {
		return &loyalty.CommandResult{Message: "Falta resolver el cliente. (Flujo completo en FID-5.)"}, nil
	}
	cfg, err := m.service.GetConfig(ctx, cmd.UserContext.CustomerID)
	if err != nil {
		return nil, err
	}
	res, err := m.service.AddStamp(ctx, AddStampReq{
		CustomerSisfiID: cfg.CustomerSisfiID,
		ClientID:        clientID,
		CollaboratorID:  cmd.UserContext.UserID,
	})
	if err != nil {
		return nil, err
	}
	msg := fmt.Sprintf("Sello sumado: %d / %d", res.StampsCount, res.CardSlots)
	if res.Completed {
		msg += "\n¡Tarjeta completada! Avisale al cliente."
	}
	return &loyalty.CommandResult{Message: msg}, nil
}

func (m *Module) handleUndoStamp(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	_, err := m.service.UndoLastStamp(ctx, cmd.UserContext.UserID)
	if err != nil {
		if errors.Is(err, ErrNoStampToUndo) {
			return &loyalty.CommandResult{Message: "No hay un sello tuyo dentro de las últimas 2 horas."}, nil
		}
		return nil, err
	}
	return &loyalty.CommandResult{Message: "Último sello deshecho."}, nil
}

func (m *Module) handleConfirmRedemption(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	// Full code-based confirmation lands with FID-5. Here we acknowledge.
	_ = cmd.Data["code"]
	return &loyalty.CommandResult{Message: "Confirmación de canje pendiente (FID-5)."}, nil
}
