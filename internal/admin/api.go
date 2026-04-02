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
	rg.POST("/login/google", h.GoogleLogin)
	rg.POST("/register", h.Register)
}

func (h *APIHandler) RegisterOnboardingRoutes(rg *gin.RouterGroup) {
	rg.POST("/register", h.Onboard)
	rg.POST("/register/google", h.GoogleOnboard)
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

func (h *APIHandler) GoogleLogin(c *gin.Context) {
	var req GoogleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "google_token es requerido"})
		return
	}

	resp, err := h.service.GoogleLogin(req.GoogleToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no se pudo autenticar con Google"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *APIHandler) GoogleOnboard(c *gin.Context) {
	var req GoogleOnboardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "google_token, nombre y telefono son requeridos"})
		return
	}

	resp, err := h.service.GoogleOnboard(req)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *APIHandler) Onboard(c *gin.Context) {
	var req OnboardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nombre, telefono, email y password son requeridos"})
		return
	}

	resp, err := h.service.Onboard(req)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}
