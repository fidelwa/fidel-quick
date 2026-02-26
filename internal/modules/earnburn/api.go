package earnburn

import (
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
		Type        string `json:"type" binding:"required"`
		Name        string `json:"name" binding:"required"`
		PointsRatio int    `json:"points_ratio"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("datos invalidos", err))
		return
	}

	p := &Program{
		CustomerID:  req.CustomerID,
		Type:        req.Type,
		Name:        req.Name,
		PointsRatio: req.PointsRatio,
	}
	if err := h.service.CreateProgram(c.Request.Context(), p); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": p.ID})
}

func (h *APIHandler) updateProgram(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		PointsRatio int    `json:"points_ratio"`
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
	p := &Program{
		ID:          c.Param("id"),
		Name:        req.Name,
		PointsRatio: req.PointsRatio,
		Active:      active,
	}
	if err := h.service.UpdateProgram(c.Request.Context(), p); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// --- Reward endpoints ---

func (h *APIHandler) createReward(c *gin.Context) {
	programID := c.Param("id")
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
	if err := h.service.CreateRewardAdmin(c.Request.Context(), programID, rw); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": rw.ID})
}

func (h *APIHandler) listRewards(c *gin.Context) {
	programID := c.Param("id")
	rewards, err := h.service.ListAllRewards(c.Request.Context(), programID)
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
	programID := c.Param("id")
	clientID := c.Param("client_id")

	balance, err := h.service.GetBalance(c.Request.Context(), clientID, programID)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"client_id": clientID, "program_id": programID, "balance": balance})
}

func (h *APIHandler) getClientTransactions(c *gin.Context) {
	programID := c.Param("id")
	clientID := c.Param("client_id")

	txs, err := h.service.ListTransactions(c.Request.Context(), clientID, programID, 50)
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
		HashID string `json:"hash_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("datos invalidos", err))
		return
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
