package earnburn

import (
	"crypto/rand"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/theluisbolivar/fidel-quick/internal/apperror"
)

// APIHandler provides REST endpoints for admin operations.
type APIHandler struct {
	service *Service
}

func NewAPIHandler(service *Service) *APIHandler {
	return &APIHandler{service: service}
}

// RegisterRoutes adds earn-burn REST API routes.
func (h *APIHandler) RegisterRoutes(rg *gin.RouterGroup) {
	programs := rg.Group("/programs")
	{
		programs.GET("", h.listPrograms)
		programs.POST("", h.createProgram)
		programs.PUT("/:id", h.updateProgram)
		programs.POST("/:id/rewards", h.createReward)
		programs.GET("/:id/rewards", h.listRewards)
		programs.PUT("/:id/rewards/:reward_id", h.updateReward)
		programs.GET("/:id/clients/:client_id/balance", h.getClientBalance)
		programs.GET("/:id/clients/:client_id/transactions", h.getClientTransactions)
	}

	customers := rg.Group("/customers")
	{
		customers.POST("", h.createCustomer)
		customers.GET("/:id", h.getCustomer)
		customers.PUT("/:id", h.updateCustomer)
		customers.POST("/:id/collaborators", h.createCollaborator)
		customers.GET("/:id/collaborators", h.listCollaborators)
		customers.GET("/:id/clients", h.listClients)
		customers.GET("/:id/feedback", h.listFeedback)
	}
}

// --- Program endpoints ---

func (h *APIHandler) listPrograms(c *gin.Context) {
	customerID := c.Query("customer_id")
	programs, err := h.service.ListPrograms(c.Request.Context(), customerID)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, programs)
}

func (h *APIHandler) createProgram(c *gin.Context) {
	var req struct {
		CustomerID  string `json:"customer_id" binding:"required"`
		Name        string `json:"name" binding:"required"`
		PointsRatio int    `json:"points_ratio" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("datos invalidos", err))
		return
	}
	p := &EarnBurnProgram{
		CustomerID:  req.CustomerID,
		Name:        req.Name,
		PointsRatio: req.PointsRatio,
	}
	if err := h.service.CreateProgram(c.Request.Context(), p); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":           p.CustomerSisfiID,
		"customer_id":  p.CustomerID,
		"name":         p.Name,
		"points_ratio": p.PointsRatio,
		"active":       true,
	})
}

// updateProgram updates program config (name, points_ratio, active) plus the
// loyalty options: expiry_days (FID-34) y min_ticket_amount (FID-36).
//
// FULL-REPLACE (LG-1): expiry_days y min_ticket_amount son full-replace en la
// capa de repositorio — un valor ausente en el JSON llega como nil y ESCRIBE NULL
// (borra el límite). El frontend por tanto debe enviar SIEMPRE ambos campos con su
// valor actual (vacío => null => sin límite), incluso al alternar `active`.
func (h *APIHandler) updateProgram(c *gin.Context) {
	var req struct {
		Name            string   `json:"name"`
		PointsRatio     int      `json:"points_ratio"`
		Active          *bool    `json:"active"`
		ExpiryDays      *int     `json:"expiry_days"`
		MinTicketAmount *float64 `json:"min_ticket_amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("datos invalidos", err))
		return
	}

	p := &EarnBurnProgram{
		CustomerSisfiID: c.Param("id"),
		Name:            req.Name,
		PointsRatio:     req.PointsRatio,
		ExpiryDays:      req.ExpiryDays,
		MinTicketAmount: req.MinTicketAmount,
	}
	if err := h.service.UpdateProgram(c.Request.Context(), p, req.Active); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// --- Reward endpoints ---

func (h *APIHandler) createReward(c *gin.Context) {
	customerSisfiID := c.Param("id")
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		PointsCost  int    `json:"points_cost" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("datos invalidos", err))
		return
	}

	rw := &Reward{
		Name:        req.Name,
		Description: req.Description,
		PointsCost:  req.PointsCost,
	}
	if err := h.service.CreateRewardAdmin(c.Request.Context(), customerSisfiID, rw); err != nil {
		c.Error(err)
		return
	}
	rw.CustomerSisfiID = customerSisfiID
	rw.Active = true
	c.JSON(http.StatusCreated, rw)
}

func (h *APIHandler) listRewards(c *gin.Context) {
	customerSisfiID := c.Param("id")
	rewards, err := h.service.ListAllRewards(c.Request.Context(), customerSisfiID)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, rewards)
}

func (h *APIHandler) updateReward(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		PointsCost  int    `json:"points_cost"`
		Active      *bool  `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("datos invalidos", err))
		return
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}
	rw := &Reward{
		ID:          c.Param("reward_id"),
		Name:        req.Name,
		Description: req.Description,
		PointsCost:  req.PointsCost,
		Active:      active,
	}
	if err := h.service.UpdateRewardAdmin(c.Request.Context(), rw); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// --- Client balance/transactions ---

func (h *APIHandler) getClientBalance(c *gin.Context) {
	customerSisfiID := c.Param("id")
	clientID := c.Param("client_id")

	balance, err := h.service.GetBalance(c.Request.Context(), clientID, customerSisfiID)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"client_id": clientID, "customer_sisfi_id": customerSisfiID, "balance": balance})
}

func (h *APIHandler) getClientTransactions(c *gin.Context) {
	customerSisfiID := c.Param("id")
	clientID := c.Param("client_id")

	txs, err := h.service.ListTransactions(c.Request.Context(), clientID, customerSisfiID, 50)
	if err != nil {
		c.Error(err)
		return
	}

	var result []map[string]interface{}
	for _, tx := range txs {
		result = append(result, map[string]interface{}{
			"id": tx.ID, "type": tx.Type, "amount": tx.Amount,
			"balance_after": tx.BalanceAfter, "created_at": tx.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, result)
}

// --- Customer endpoints ---

func (h *APIHandler) createCustomer(c *gin.Context) {
	var req struct {
		Name           string `json:"name" binding:"required"`
		Slug           string `json:"slug" binding:"required"`
		Phone          string `json:"phone" binding:"required"`
		Address        string `json:"address"`
		LogoURL        string `json:"logo_url"`
		Description    string `json:"description"`
		WelcomeMessage string `json:"welcome_message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("datos invalidos", err))
		return
	}

	cust := &Customer{
		Name:           req.Name,
		Slug:           req.Slug,
		Phone:          req.Phone,
		Address:        req.Address,
		LogoURL:        req.LogoURL,
		Description:    req.Description,
		WelcomeMessage: req.WelcomeMessage,
	}
	if err := h.service.CreateCustomer(c.Request.Context(), cust); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": cust.ID})
}

func (h *APIHandler) getCustomer(c *gin.Context) {
	cust, err := h.service.GetCustomer(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, cust)
}

func (h *APIHandler) updateCustomer(c *gin.Context) {
	var req struct {
		Name           string `json:"name"`
		Phone          string `json:"phone"`
		Address        string `json:"address"`
		LogoURL        string `json:"logo_url"`
		Description    string `json:"description"`
		WelcomeMessage string `json:"welcome_message"`
		Active         *bool  `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("datos invalidos", err))
		return
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}
	cust := &Customer{
		ID:             c.Param("id"),
		Name:           req.Name,
		Phone:          req.Phone,
		Address:        req.Address,
		LogoURL:        req.LogoURL,
		Description:    req.Description,
		WelcomeMessage: req.WelcomeMessage,
		Active:         active,
	}
	if err := h.service.UpdateCustomer(c.Request.Context(), cust); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// --- Collaborator endpoints ---

func (h *APIHandler) createCollaborator(c *gin.Context) {
	customerID := c.Param("id")
	var req struct {
		Name   string `json:"name" binding:"required"`
		Phone  string `json:"phone" binding:"required"`
		HashID string `json:"hash_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("datos invalidos", err))
		return
	}

	if req.HashID == "" {
		buf := make([]byte, 6)
		rand.Read(buf)
		req.HashID = fmt.Sprintf("%x", buf)
	}

	collab := &Collaborator{
		CustomerID: customerID,
		Name:       req.Name,
		Phone:      req.Phone,
		HashID:     req.HashID,
	}
	if err := h.service.CreateCollaborator(c.Request.Context(), collab); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": collab.ID})
}

func (h *APIHandler) listCollaborators(c *gin.Context) {
	collabs, err := h.service.ListCollaborators(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, collabs)
}

// --- Clients ---

func (h *APIHandler) listClients(c *gin.Context) {
	clients, err := h.service.ListClients(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, clients)
}

// --- Feedback ---

func (h *APIHandler) listFeedback(c *gin.Context) {
	entries, err := h.service.ListFeedback(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, entries)
}

