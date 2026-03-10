package whatsapp

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/theluisbolivar/fidel-quick/internal/flow"
	"github.com/theluisbolivar/fidel-quick/internal/loyalty"
	"github.com/theluisbolivar/fidel-quick/internal/resolver"
	"github.com/theluisbolivar/fidel-quick/internal/session"
)

// WebhookHandler processes incoming WhatsApp messages through the full pipeline.
type WebhookHandler struct {
	verifyToken string
	client      *Client
	session     *session.Manager
	business    *resolver.BusinessResolver
	role        *resolver.RoleResolver
	repo        resolver.Repository
	engine      *flow.Engine
	log         *slog.Logger
}

func NewWebhookHandler(
	verifyToken string,
	client *Client,
	sess *session.Manager,
	business *resolver.BusinessResolver,
	role *resolver.RoleResolver,
	repo resolver.Repository,
	engine *flow.Engine,
	log *slog.Logger,
) *WebhookHandler {
	return &WebhookHandler{
		verifyToken: verifyToken,
		client:      client,
		session:     sess,
		business:    business,
		role:        role,
		repo:        repo,
		engine:      engine,
		log:         log,
	}
}

// Verify handles GET /webhook — WhatsApp verification challenge.
func (h *WebhookHandler) Verify(c *gin.Context) {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if mode == "subscribe" && token == h.verifyToken {
		h.log.Info("webhook verified")
		c.String(http.StatusOK, challenge)
		return
	}
	c.String(http.StatusForbidden, "Forbidden")
}

// Receive handles POST /webhook — incoming messages.
func (h *WebhookHandler) Receive(c *gin.Context) {
	var payload WebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		h.log.Error("invalid webhook payload", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Always respond 200 quickly to Meta
	c.JSON(http.StatusOK, gin.H{"status": "received"})

	// Process messages asynchronously to avoid webhook timeout.
	// Use a detached context since the HTTP request context is already done.
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}
			for _, msg := range change.Value.Messages {
				go h.processMessage(context.Background(), msg)
			}
		}
	}
}

func (h *WebhookHandler) processMessage(ctx context.Context, msg Message) {
	phone := msg.From
	msgType := msg.Type
	msgText := ""
	imageID := ""

	switch msgType {
	case "text":
		if msg.Text != nil {
			msgText = msg.Text.Body
		}
	case "image":
		if msg.Image != nil {
			imageID = msg.Image.ID
		}
	case "interactive":
		if msg.Interactive != nil && msg.Interactive.ListReply != nil {
			msgType = "interactive"
			msgText = msg.Interactive.ListReply.ID
		}
	default:
		h.log.Info("ignoring unsupported message type", "type", msgType, "from", phone)
		return
	}

	h.log.Info("message received", "from", phone, "type", msgType)

	// 1. Check for active session
	uc, err := h.session.GetSession(ctx, phone)
	if err != nil {
		h.log.Error("get session failed", "error", err)
		return
	}

	// 2. No session — resolve business
	if uc == nil {
		h.log.Info("no session found, resolving context", "phone", phone)
		uc, err = h.resolveContext(ctx, phone, msgText)
		if err != nil {
			h.log.Error("resolve context failed", "error", err, "phone", phone)
			return
		}
		if uc == nil {
			h.log.Info("resolve context returned nil (pending selection or not registered)", "phone", phone)
			return
		}
		// Session just created — show main menu instead of forwarding the original message
		msgType = "text"
		msgText = ""
	}

	// 2b. Backfill ActiveModules for stale sessions
	if len(uc.ActiveModules) == 0 {
		modules, err := h.repo.GetActiveProgramTypes(ctx, uc.CustomerID)
		if err != nil {
			h.log.Error("backfill active modules failed", "error", err)
		} else {
			uc.ActiveModules = modules
			h.session.SetSession(ctx, phone, uc)
		}
	}

	// 3. Intercept "cambiar_negocio" — reset session and re-resolve
	if msgType == "interactive" && msgText == "cambiar_negocio" {
		h.session.DeleteSession(ctx, phone)
		h.engine.ResetFlow(ctx, phone, uc.CustomerID)
		if _, err := h.resolveContext(ctx, phone, ""); err != nil {
			h.log.Error("re-resolve after cambiar_negocio failed", "error", err)
		}
		return
	}

	// 4. Build user context for flow engine
	user := loyalty.UserContext{
		CustomerID:    uc.CustomerID,
		BusinessName:  uc.BusinessName,
		Role:          uc.Role,
		UserID:        uc.UserID,
		Phone:         phone,
		ActiveModules: uc.ActiveModules,
	}

	// 5. Resolve image URL if needed
	imageURL := ""
	if imageID != "" {
		mediaURL, err := h.client.GetMediaURL(ctx, imageID)
		if err != nil {
			h.log.Error("get media url failed", "error", err)
		} else {
			imageURL = mediaURL
		}
	}

	// 6. Dispatch to flow engine
	if err := h.engine.HandleMessage(ctx, user, msgType, msgText, imageURL); err != nil {
		h.log.Error("flow engine error", "error", err, "phone", phone)
		h.client.SendText(ctx, phone, "Ocurrio un error. Intenta de nuevo.")
	}
}

// resolveContext handles business + role resolution when there's no active session.
func (h *WebhookHandler) resolveContext(ctx context.Context, phone, msgText string) (*session.UserContext, error) {
	// Check pending business selection first
	pending, err := h.session.GetPendingSelection(ctx, phone)
	if err != nil {
		return nil, err
	}
	if pending != nil {
		h.log.Info("pending business selection found", "phone", phone, "options", len(pending))
		// User is responding to a business selection prompt
		for _, opt := range pending {
			if opt.CustomerID == msgText || opt.Name == msgText {
				h.session.DeletePendingSelection(ctx, phone)
				return h.resolveAndCreateSession(ctx, phone, opt.CustomerID, opt.Name)
			}
		}
		// Invalid selection — re-prompt
		h.client.SendText(ctx, phone, "Seleccion invalida. Intenta de nuevo.")
		return nil, nil
	}

	// Resolve business from deeplink or phone
	biz, multi, err := h.business.Resolve(ctx, phone, msgText)
	if err != nil {
		return nil, err
	}

	if multi != nil {
		h.log.Info("multiple businesses found, prompting selection", "phone", phone, "count", len(multi.Options))

		var options []ListOption
		for _, opt := range multi.Options {
			options = append(options, ListOption{
				ID:    opt.CustomerID,
				Title: opt.Name,
			})
		}
		if err := h.client.SendInteractiveList(ctx, phone, "Seleccionar negocio", "En cual negocio quieres operar?", options); err != nil {
			h.log.Error("failed to send business selection list", "error", err, "phone", phone)
			// Don't store pending selection if the list wasn't delivered
			return nil, nil
		}
		// Only persist after successful send
		h.session.SetPendingSelection(ctx, phone, multi.Options)
		return nil, nil
	}

	if biz == nil {
		// Not registered
		h.log.Info("user not registered in any business", "phone", phone)
		if err := h.client.SendText(ctx, phone, "Hola! Aun no estas registrado.\nEscanea el codigo QR en el establecimiento para unirte al programa de fidelidad."); err != nil {
			h.log.Error("failed to send not-registered message", "error", err, "phone", phone)
		}
		return nil, nil
	}

	// Single business found (or deeplink) — auto-register if new
	if biz.IsNew {
		if err := h.autoRegisterClient(ctx, phone, biz.CustomerID); err != nil {
			h.log.Error("auto-register client failed", "error", err, "phone", phone)
			return nil, err
		}
	}

	return h.resolveAndCreateSession(ctx, phone, biz.CustomerID, biz.BusinessName)
}

func (h *WebhookHandler) resolveAndCreateSession(ctx context.Context, phone, customerID, businessName string) (*session.UserContext, error) {
	// Clear any stale flow state from a previous session
	h.engine.ResetFlow(ctx, phone, customerID)

	roleResult, err := h.role.Resolve(ctx, phone, customerID)
	if err != nil {
		return nil, err
	}

	if roleResult == nil {
		h.log.Warn("role not resolved", "phone", phone, "customer", customerID)
		h.client.SendText(ctx, phone, "No se pudo determinar tu rol. Contacta al negocio.")
		return nil, nil
	}

	modules, err := h.repo.GetActiveProgramTypes(ctx, customerID)
	if err != nil {
		h.log.Error("get active program types failed", "error", err, "customer", customerID)
	}

	uc := &session.UserContext{
		CustomerID:    customerID,
		Role:          roleResult.Role,
		UserID:        roleResult.UserID,
		BusinessName:  businessName,
		ActiveModules: modules,
	}

	if err := h.session.SetSession(ctx, phone, uc); err != nil {
		return nil, err
	}

	return uc, nil
}

// autoRegisterClient inserts a new client record when a user arrives via deeplink.
func (h *WebhookHandler) autoRegisterClient(ctx context.Context, phone, customerID string) error {
	return h.repo.RegisterClient(ctx, customerID, phone)
}
