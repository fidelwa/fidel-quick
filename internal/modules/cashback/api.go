package cashback

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/theluisbolivar/fidel-quick/internal/apperror"
)

// APIHandler provides REST endpoints for cashback admin operations.
type APIHandler struct {
	service *Service
}

func NewAPIHandler(service *Service) *APIHandler {
	return &APIHandler{service: service}
}

// RegisterRoutes adds cashback REST API routes.
func (h *APIHandler) RegisterRoutes(rg *gin.RouterGroup) {
	programs := rg.Group("/cashback-programs")
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
		CustomerID   string  `json:"customer_id" binding:"required"`
		Name         string  `json:"name" binding:"required"`
		CashbackRate float64 `json:"cashback_rate"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("datos invalidos", err))
		return
	}

	p := &CashbackProgram{
		CustomerID:   req.CustomerID,
		Type:         "cashback",
		Name:         req.Name,
		CashbackRate: req.CashbackRate,
	}
	if err := h.service.CreateProgram(c.Request.Context(), p); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": p.ID})
}

func (h *APIHandler) updateProgram(c *gin.Context) {
	var req struct {
		Name         string  `json:"name"`
		CashbackRate float64 `json:"cashback_rate"`
		Active       *bool   `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("datos invalidos", err))
		return
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}
	p := &CashbackProgram{
		ID:           c.Param("id"),
		Name:         req.Name,
		CashbackRate: req.CashbackRate,
		Active:       active,
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
		Name        string  `json:"name" binding:"required"`
		Description string  `json:"description"`
		Cost        float64 `json:"cost" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("datos invalidos", err))
		return
	}

	rw := &CashbackReward{
		Name:        req.Name,
		Description: req.Description,
		Cost:        req.Cost,
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
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Cost        float64 `json:"cost"`
		Active      *bool   `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("datos invalidos", err))
		return
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}
	rw := &CashbackReward{
		ID:          c.Param("reward_id"),
		Name:        req.Name,
		Description: req.Description,
		Cost:        req.Cost,
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
