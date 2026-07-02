package pushcard

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

// handleRedeem generates a redemption code for a completed card. The
// collaborator confirms by entering this code (handled in pc_confirm_redemption).
func (m *Module) handleRedeem(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	cfg, err := m.service.GetConfig(ctx, cmd.UserContext.CustomerID)
	if err != nil {
		return nil, err
	}
	progress, err := m.service.GetProgress(ctx, cfg.CustomerSisfiID, cmd.UserContext.UserID)
	if err != nil {
		return nil, err
	}
	// Open card not yet complete — still progressing.
	if progress.HasOpenCard && progress.StampsCount < progress.CardSlots {
		msg := fmt.Sprintf("Aún no completás la tarjeta de *%s*.\n%s\nTe faltan %d sellos.",
			cfg.Name, progress.Visual, progress.CardSlots-progress.StampsCount)
		return &loyalty.CommandResult{Message: msg}, nil
	}

	// Either there's a completed card waiting or no card at all.
	// Preferimos la recompensa configurada (reward_on_complete, texto libre
	// definido en el wizard/panel); si no hay, caemos al nombre del programa.
	rewardName := cfg.RewardOnComplete
	if strings.TrimSpace(rewardName) == "" {
		rewardName = cfg.Name
	}
	code, err := m.service.RequestRedemption(ctx,
		cfg.CustomerSisfiID, cmd.UserContext.UserID, cmd.UserContext.CustomerID, rewardName)
	if err != nil {
		return &loyalty.CommandResult{Message: err.Error()}, nil
	}

	msg := fmt.Sprintf("¡Felicitaciones! Tu tarjeta está completa.\n\nTu código de canje: *%s*\nVálido por 1 hora. Mostrale el código al colaborador para canjear *%s*.",
		code, rewardName)
	return &loyalty.CommandResult{Message: msg}, nil
}

// handleAddStamp resolves the client by the phone the collaborator typed, then
// adds a stamp to that client's pushcard. If the client doesn't belong to the
// business, the bot asks them to scan the QR.
func (m *Module) handleAddStamp(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	cfg, err := m.service.GetConfig(ctx, cmd.UserContext.CustomerID)
	if err != nil {
		return nil, err
	}

	phone := strings.TrimSpace(cmd.Data["client_phone"])
	clientID, err := m.service.FindClientIDByPhone(ctx, cmd.UserContext.CustomerID, phone)
	if err != nil {
		return nil, err
	}
	if clientID == "" {
		msg := "El cliente no está registrado en este negocio.\nPedile que escanee el QR para unirse."
		return &loyalty.CommandResult{Message: msg}, nil
	}

	res, err := m.service.AddStamp(ctx, AddStampReq{
		CustomerSisfiID: cfg.CustomerSisfiID,
		ClientID:        clientID,
		CollaboratorID:  cmd.UserContext.UserID,
	})
	if err != nil {
		return nil, err
	}

	visual := buildVisual(res.StampsCount, res.CardSlots)
	msg := fmt.Sprintf("Sello sumado.\n%s\n%d / %d", visual, res.StampsCount, res.CardSlots)
	if res.Completed {
		completedReward := cfg.RewardOnComplete
		if strings.TrimSpace(completedReward) == "" {
			completedReward = cfg.Name
		}
		msg += fmt.Sprintf("\n\n¡Tarjeta completada! Avisale al cliente para que canjee *%s*.", completedReward)
	}
	msg += "\n\n_Podés deshacer este sello en las próximas 2 horas._"
	return &loyalty.CommandResult{Message: msg}, nil
}

func (m *Module) handleUndoStamp(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	_, err := m.service.UndoLastStamp(ctx, cmd.UserContext.UserID)
	if err != nil {
		if errors.Is(err, ErrNoStampToUndo) {
			return &loyalty.CommandResult{Message: "No hay un sello tuyo dentro de las últimas 2 horas."}, nil
		}
		if errors.Is(err, ErrCardCancelled) {
			return &loyalty.CommandResult{Message: "Esa tarjeta venció y fue cancelada; no se puede deshacer el sello."}, nil
		}
		return nil, err
	}
	return &loyalty.CommandResult{Message: "Último sello deshecho."}, nil
}

func (m *Module) handleConfirmRedemption(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	code := strings.TrimSpace(cmd.Data["code"])
	data, err := m.service.ConfirmRedemption(ctx, code, cmd.UserContext.UserID)
	if err != nil {
		return &loyalty.CommandResult{Message: err.Error()}, nil
	}
	rewardName := data.RewardName
	if rewardName == "" {
		rewardName = "tarjeta de sellos"
	}
	msg := fmt.Sprintf("Canje confirmado.\nEntregá: *%s*", rewardName)
	return &loyalty.CommandResult{Message: msg}, nil
}
