package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIHandler struct {
	service *Service
}

func NewAPIHandler(service *Service) *APIHandler {
	return &APIHandler{service: service}
}

func (h *APIHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/login", h.Login)
	rg.POST("/register", h.Register)
}

func (h *APIHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email y password son requeridos"})
		return
	}

	resp, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "credenciales invalidas"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *APIHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email, password y customer_id son requeridos"})
		return
	}

	resp, err := h.service.Register(req.Email, req.Password, req.CustomerID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}
