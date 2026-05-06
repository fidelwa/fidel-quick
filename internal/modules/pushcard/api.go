package pushcard

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/theluisbolivar/fidel-quick/internal/apperror"
)

// APIHandler exposes admin REST endpoints for pushcard.
type APIHandler struct {
	service *Service
}

func NewAPIHandler(service *Service) *APIHandler {
	return &APIHandler{service: service}
}

// RegisterRoutes wires the pushcard endpoints under /pushcard.
//
//	GET  /pushcard/config?customer_id=...
//	POST /pushcard/programs                — activates the pushcard sisfi for a customer
//	PUT  /pushcard/programs/:id/config
//	GET  /pushcard/programs/:id/cards?status=open|completed&limit=50
func (h *APIHandler) RegisterRoutes(rg *gin.RouterGroup) {
	pc := rg.Group("/pushcard")
	{
		pc.GET("/config", h.getConfigByCustomer)
		pc.POST("/programs", h.createProgram)
		pc.PUT("/programs/:id/config", h.upsertConfig)
		pc.GET("/programs/:id/cards", h.listCards)
	}
}

// createProgram activates the pushcard sisfi for a customer in one shot:
// creates the customer_sisfi and an initial pushcard_config (slots only,
// reward deferred). Used by the onboarding wizard.
func (h *APIHandler) createProgram(c *gin.Context) {
	var req struct {
		CustomerID string `json:"customer_id" binding:"required"`
		Name       string `json:"name"`
		CardSlots  int    `json:"card_slots" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("datos invalidos", err))
		return
	}
	if req.CardSlots <= 0 || req.CardSlots > 50 {
		c.Error(apperror.BadRequest("card_slots debe estar entre 1 y 50", nil))
		return
	}

	cfg, err := h.service.CreateProgram(c.Request.Context(), req.CustomerID, req.Name, req.CardSlots)
	if err != nil {
		if errors.Is(err, ErrProgramAlreadyExists) {
			c.Error(apperror.Conflict("el negocio ya tiene un programa pushcard activo", err))
			return
		}
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, cfg)
}

// getConfigByCustomer returns the active pushcard config for a customer.
func (h *APIHandler) getConfigByCustomer(c *gin.Context) {
	customerID := c.Query("customer_id")
	if customerID == "" {
		c.Error(apperror.BadRequest("customer_id requerido", nil))
		return
	}
	cfg, err := h.service.GetConfig(c.Request.Context(), customerID)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// upsertConfig validates and persists the pushcard config for a customer_sisfi.
func (h *APIHandler) upsertConfig(c *gin.Context) {
	customerSisfiID := c.Param("id")
	var req struct {
		CardSlots        int    `json:"card_slots" binding:"required"`
		RewardOnComplete string `json:"reward_on_complete"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("datos invalidos", err))
		return
	}
	if req.CardSlots <= 0 {
		c.Error(apperror.BadRequest("card_slots debe ser mayor a 0", nil))
		return
	}

	cfg := &Config{
		CustomerSisfiID:  customerSisfiID,
		CardSlots:        req.CardSlots,
		RewardOnComplete: req.RewardOnComplete,
	}
	if err := h.service.UpsertConfig(c.Request.Context(), cfg); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// listCards returns recent pushcards for a customer_sisfi, optionally filtered
// by status.
func (h *APIHandler) listCards(c *gin.Context) {
	customerSisfiID := c.Param("id")
	status := c.Query("status")
	limit := 50
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	cards, err := h.service.ListCards(c.Request.Context(), customerSisfiID, status, limit)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, cards)
}
