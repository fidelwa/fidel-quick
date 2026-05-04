package sisfi

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

func (h *APIHandler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.GET("/sisfi", h.ListSisfi)
}

func (h *APIHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/customer-sisfi", h.ListByCustomer)
	rg.POST("/customer-sisfi", h.Create)
	rg.PUT("/customer-sisfi/:id", h.Update)
}

func (h *APIHandler) ListSisfi(c *gin.Context) {
	items, err := h.repo.ListSisfi()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al listar sistemas"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *APIHandler) ListByCustomer(c *gin.Context) {
	customerID := c.Query("customer_id")
	if customerID == "" {
		customerID = c.GetString("customer_id")
	}
	if customerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_id requerido"})
		return
	}
	items, err := h.repo.ListByCustomer(customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al listar sistemas del customer"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *APIHandler) Create(c *gin.Context) {
	var req struct {
		CustomerID string `json:"customer_id" binding:"required"`
		SisfiID    string `json:"sisfi_id" binding:"required"`
		Name       string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_id, sisfi_id y name son requeridos"})
		return
	}

	cs := &CustomerSisfi{
		CustomerID: req.CustomerID,
		SisfiID:    req.SisfiID,
		Name:       req.Name,
		Active:     true,
	}
	if err := h.repo.Create(cs); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "sistema ya existe para este customer"})
		return
	}
	c.JSON(http.StatusCreated, cs)
}

func (h *APIHandler) Update(c *gin.Context) {
	id := c.Param("id")
	cs, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "sistema no encontrado"})
		return
	}

	var req struct {
		Name   string `json:"name"`
		Active *bool  `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos"})
		return
	}

	if req.Name != "" {
		cs.Name = req.Name
	}
	if req.Active != nil {
		cs.Active = *req.Active
	}

	if err := h.repo.Update(cs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al actualizar"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}
