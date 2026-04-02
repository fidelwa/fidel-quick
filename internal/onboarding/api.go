package onboarding

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIHandler struct {
	repo *Repository
}

func NewAPIHandler(repo *Repository) *APIHandler {
	return &APIHandler{repo: repo}
}

func (h *APIHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/onboarding", h.Get)
	rg.PUT("/onboarding/step", h.UpdateStep)
	rg.POST("/onboarding/complete", h.Complete)
}

func (h *APIHandler) Get(c *gin.Context) {
	customerID := c.GetString("customer_id")
	if customerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_id requerido"})
		return
	}

	o, err := h.repo.GetByCustomerID(customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al consultar onboarding"})
		return
	}
	if o == nil {
		c.JSON(http.StatusOK, gin.H{"current_step": 1, "completed": false})
		return
	}
	c.JSON(http.StatusOK, o)
}

func (h *APIHandler) UpdateStep(c *gin.Context) {
	customerID := c.GetString("customer_id")
	if customerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_id requerido"})
		return
	}

	var req struct {
		Step int `json:"step" binding:"required,min=1,max=4"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "step invalido (1-4)"})
		return
	}

	o, err := h.repo.Upsert(customerID, req.Step)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al actualizar onboarding"})
		return
	}
	c.JSON(http.StatusOK, o)
}

func (h *APIHandler) Complete(c *gin.Context) {
	customerID := c.GetString("customer_id")
	if customerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_id requerido"})
		return
	}

	o, err := h.repo.Complete(customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al completar onboarding"})
		return
	}
	c.JSON(http.StatusOK, o)
}
