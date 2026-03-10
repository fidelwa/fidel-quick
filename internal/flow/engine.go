package flow

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/theluisbolivar/fidel-quick/internal/loyalty"
)

// cmdLogAttrs returns slog attributes for a failed command, including full user
// context and the collected flow data so that every error log is self-contained.
func cmdLogAttrs(commandID string, user loyalty.UserContext, data map[string]string, err error) []any {
	attrs := []any{
		"command", commandID,
		"error", err,
		"user_id", user.UserID,
		"role", user.Role,
		"phone", user.Phone,
		"customer_id", user.CustomerID,
	}
	if len(data) > 0 {
		attrs = append(attrs, "data", data)
	}
	return attrs
}

// MessageSender is the interface for sending WhatsApp messages.
// Satisfied by the WhatsApp client (Phase 6).
type MessageSender interface {
	SendText(ctx context.Context, to, text string) error
	SendInteractiveList(ctx context.Context, to, header, body string, options []ListOption) error
}

// ListOption represents a single item in a WhatsApp interactive list.
type ListOption struct {
	ID          string
	Title       string
	Description string
}

// PhotoProcessor handles invoice photo analysis and storage.
// When nil, photos are stored as raw URLs (backward compat).
type PhotoProcessor interface {
	ProcessPhoto(ctx context.Context, imageURL string) (*PhotoProcessResult, error)
}

// PhotoProcessResult from processing an invoice photo.
type PhotoProcessResult struct {
	StorageURL string
	Amount     float64
	Currency   string
}

// FlowStore persists flow state. Satisfied by *StateStore (Redis).
type FlowStore interface {
	Get(ctx context.Context, phone, customerID string) (*State, error)
	Set(ctx context.Context, phone, customerID string, fs *State) error
	Delete(ctx context.Context, phone, customerID string) error
}

// Engine manages step-by-step flows and menu presentation.
type Engine struct {
	registry       *loyalty.Registry
	store          FlowStore
	sender         MessageSender
	photoProcessor PhotoProcessor
	log            *slog.Logger
}

func NewEngine(registry *loyalty.Registry, store FlowStore, sender MessageSender, photoProcessor PhotoProcessor, log *slog.Logger) *Engine {
	return &Engine{
		registry:       registry,
		store:          store,
		sender:         sender,
		photoProcessor: photoProcessor,
		log:            log,
	}
}

// HandleMessage processes an incoming message within the user's context.
// It checks for active flows, menu selections, or presents the main menu.
func (e *Engine) HandleMessage(ctx context.Context, user loyalty.UserContext, msgType, msgText, imageURL string) error {
	// 1. Check for active flow
	flowState, err := e.store.Get(ctx, user.Phone, user.CustomerID)
	if err != nil {
		return fmt.Errorf("get flow state: %w", err)
	}

	if flowState != nil {
		return e.processStep(ctx, user, flowState, msgType, msgText, imageURL)
	}

	// 2. Check if this is a menu selection (interactive list reply)
	if msgType == "interactive" {
		return e.handleMenuSelection(ctx, user, msgText)
	}

	// 3. Text libre — present main menu
	return e.presentMainMenu(ctx, user)
}

// processStep handles the user's response to the current flow step.
func (e *Engine) processStep(ctx context.Context, user loyalty.UserContext, fs *State, msgType, input, imageURL string) error {
	flowDef, ok := e.registry.GetFlowDefinition(fs.CurrentFlow)
	if !ok {
		// Flow definition not found, clear state and show menu
		e.store.Delete(ctx, user.Phone, user.CustomerID)
		return e.presentMainMenu(ctx, user)
	}

	step := flowDef.Steps[fs.CurrentStep]

	// Use image URL if step needs photo
	actualInput := input
	if step.NeedsPhoto {
		if imageURL == "" {
			return e.sender.SendText(ctx, user.Phone, "Envia una foto para continuar.")
		}
		if e.photoProcessor != nil {
			result, err := e.photoProcessor.ProcessPhoto(ctx, imageURL)
			if err != nil {
				e.log.Error("photo processing failed", "error", err)
				return e.sender.SendText(ctx, user.Phone, "Error al procesar la foto. Intenta de nuevo.")
			}
			actualInput = result.StorageURL
			fs.CollectedData["amount"] = fmt.Sprintf("%.2f", result.Amount)
			fs.CollectedData["currency"] = result.Currency
		} else {
			actualInput = imageURL
		}
	}

	// Validate
	if step.Validate != nil {
		if err := step.Validate(actualInput); err != nil {
			return e.sender.SendText(ctx, user.Phone, fmt.Sprintf("Dato invalido: %s\n\n%s", err.Error(), step.Prompt))
		}
	}

	// Store collected data
	fs.CollectedData[step.Key] = actualInput
	fs.CurrentStep++

	// Check if flow is complete
	if fs.CurrentStep >= len(flowDef.Steps) {
		// Execute the command with collected data
		e.store.Delete(ctx, user.Phone, user.CustomerID)

		cmd := loyalty.Command{
			ID:          fs.CurrentFlow,
			UserContext:  user,
			Data:        fs.CollectedData,
		}

		result, err := e.registry.Dispatch(ctx, cmd)
		if err != nil {
			e.log.Error("command failed", cmdLogAttrs(fs.CurrentFlow, user, fs.CollectedData, err)...)
			return e.sender.SendText(ctx, user.Phone, "Ocurrio un error. Intenta de nuevo.")
		}

		return e.sendResult(ctx, user, result)
	}

	// Save updated state and prompt next step
	if err := e.store.Set(ctx, user.Phone, user.CustomerID, fs); err != nil {
		return fmt.Errorf("save flow state: %w", err)
	}

	nextStep := flowDef.Steps[fs.CurrentStep]
	return e.sender.SendText(ctx, user.Phone, nextStep.Prompt)
}

// handleMenuSelection starts a new flow or executes a direct command.
func (e *Engine) handleMenuSelection(ctx context.Context, user loyalty.UserContext, commandID string) error {
	// Check for prefixed selections (e.g. "reward:{id}" from reward list)
	if rewardID, ok := strings.CutPrefix(commandID, "reward:"); ok {
		return e.startFlowWithData(ctx, user, "request_redemption", map[string]string{
			"reward_id": rewardID,
		})
	}

	// Check for benefit prefix (cashback module)
	if benefitID, ok := strings.CutPrefix(commandID, "benefit:"); ok {
		return e.startFlowWithData(ctx, user, "cb_request_redemption", map[string]string{
			"reward_id": benefitID,
		})
	}

	// Check if this command has a flow
	flowDef, hasFlow := e.registry.GetFlowDefinition(commandID)

	if !hasFlow {
		// Direct command — execute immediately
		cmd := loyalty.Command{
			ID:          commandID,
			UserContext:  user,
			Data:        make(map[string]string),
		}

		result, err := e.registry.Dispatch(ctx, cmd)
		if err != nil {
			e.log.Error("direct command failed", cmdLogAttrs(commandID, user, cmd.Data, err)...)
			return e.sender.SendText(ctx, user.Phone, "Ocurrio un error. Intenta de nuevo.")
		}

		return e.sendResult(ctx, user, result)
	}

	// Start new flow
	fs := &State{
		CurrentFlow:   commandID,
		CurrentStep:   0,
		CollectedData: make(map[string]string),
		StartedAt:     time.Now(),
	}

	if err := e.store.Set(ctx, user.Phone, user.CustomerID, fs); err != nil {
		return fmt.Errorf("save new flow: %w", err)
	}

	firstStep := flowDef.Steps[0]
	return e.sender.SendText(ctx, user.Phone, firstStep.Prompt)
}

// startFlowWithData begins a flow with pre-collected data and advances to the next pending step.
func (e *Engine) startFlowWithData(ctx context.Context, user loyalty.UserContext, commandID string, data map[string]string) error {
	flowDef, hasFlow := e.registry.GetFlowDefinition(commandID)
	if !hasFlow {
		// No flow — execute as direct command with provided data
		cmd := loyalty.Command{
			ID:          commandID,
			UserContext:  user,
			Data:        data,
		}
		result, err := e.registry.Dispatch(ctx, cmd)
		if err != nil {
			e.log.Error("command failed", cmdLogAttrs(commandID, user, data, err)...)
			return e.sender.SendText(ctx, user.Phone, "Ocurrio un error. Intenta de nuevo.")
		}
		return e.sendResult(ctx, user, result)
	}

	// Find first step whose key is NOT already in data
	startStep := 0
	for i, step := range flowDef.Steps {
		if _, ok := data[step.Key]; ok {
			startStep = i + 1
		} else {
			break
		}
	}

	// All steps satisfied — execute immediately
	if startStep >= len(flowDef.Steps) {
		cmd := loyalty.Command{
			ID:          commandID,
			UserContext:  user,
			Data:        data,
		}
		result, err := e.registry.Dispatch(ctx, cmd)
		if err != nil {
			e.log.Error("command failed", cmdLogAttrs(commandID, user, data, err)...)
			return e.sender.SendText(ctx, user.Phone, "Ocurrio un error. Intenta de nuevo.")
		}
		return e.sendResult(ctx, user, result)
	}

	// Save flow state starting at the next pending step
	fs := &State{
		CurrentFlow:   commandID,
		CurrentStep:   startStep,
		CollectedData: data,
		StartedAt:     time.Now(),
	}

	if err := e.store.Set(ctx, user.Phone, user.CustomerID, fs); err != nil {
		return fmt.Errorf("save new flow: %w", err)
	}

	nextStep := flowDef.Steps[startStep]
	return e.sender.SendText(ctx, user.Phone, nextStep.Prompt)
}

// sendResult sends a command result to the user (text, interactive list, or both)
// then shows the main menu if no options are provided.
func (e *Engine) sendResult(ctx context.Context, user loyalty.UserContext, result *loyalty.CommandResult) error {
	// If result has options, send text first (if any), then the interactive list
	if len(result.Options) > 0 {
		if result.Message != "" {
			if err := e.sender.SendText(ctx, user.Phone, result.Message); err != nil {
				return err
			}
		}
		header := result.ListHeader
		if header == "" {
			header = "Selecciona una opcion"
		}
		var opts []ListOption
		for _, o := range result.Options {
			opts = append(opts, ListOption{ID: o.ID, Title: o.Title, Description: o.Description})
		}
		return e.sender.SendInteractiveList(ctx, user.Phone, header, "Selecciona:", opts)
	}

	// Plain text result — send and show main menu
	if err := e.sender.SendText(ctx, user.Phone, result.Message); err != nil {
		return err
	}
	return e.presentMainMenu(ctx, user)
}

// ResetFlow clears any active flow state for a user in a business.
func (e *Engine) ResetFlow(ctx context.Context, phone, customerID string) {
	e.store.Delete(ctx, phone, customerID)
}

// presentMainMenu sends the role-appropriate menu to the user.
func (e *Engine) presentMainMenu(ctx context.Context, user loyalty.UserContext) error {
	menus := e.registry.FilteredMenus(user.Role, user.ActiveModules)
	if len(menus) == 0 {
		return e.sender.SendText(ctx, user.Phone, "No hay opciones disponibles.")
	}

	var options []ListOption
	for _, m := range menus {
		options = append(options, ListOption{
			ID:          m.ID,
			Title:       m.Title,
			Description: m.Description,
		})
	}

	options = append(options, ListOption{
		ID:          "cambiar_negocio",
		Title:       "Usar otro establecimiento",
		Description: "Cambiar de negocio",
	})

	header := fmt.Sprintf("Menu principal — %s", user.BusinessName)
	body := "Selecciona una opcion:"
	return e.sender.SendInteractiveList(ctx, user.Phone, header, body, options)
}
