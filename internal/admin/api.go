package admin

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/theluisbolivar/fidel-quick/internal/apperror"
)

// FlagResolver resolves the enabled feature flags for a customer. It is
// implemented by *featureflags.Service; kept as a local interface so the admin
// package does not import featureflags (avoids a hard dependency and eases
// testing). A nil resolver simply omits flags from the /auth/me response.
type FlagResolver interface {
	EnabledFor(ctx context.Context, customerID string) (map[string]bool, error)
}

type APIHandler struct {
	service *Service
	flags   FlagResolver
}

func NewAPIHandler(service *Service) *APIHandler {
	return &APIHandler{service: service}
}

// WithFlags attaches a feature-flag resolver so /auth/me includes the flags
// active for the caller's customer (used for UI gating).
func (h *APIHandler) WithFlags(fr FlagResolver) *APIHandler {
	h.flags = fr
	return h
}

// RegisterRoutes registers public auth endpoints (login, register, login/google,
// forgot/reset password).
func (h *APIHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/login", h.Login)
	rg.POST("/login/google", h.GoogleLogin)
	rg.POST("/register", h.Register)
	rg.POST("/forgot-password", h.ForgotPassword)
	rg.POST("/reset-password", h.ResetPassword)
}

// RegisterAuthenticatedRoutes registers endpoints that require a valid admin
// JWT (account linking with Google). Mount under the JWT-protected group.
func (h *APIHandler) RegisterAuthenticatedRoutes(rg *gin.RouterGroup) {
	rg.POST("/auth/link/google", h.LinkGoogle)
	rg.DELETE("/auth/link/google", h.UnlinkGoogle)
	rg.GET("/auth/me", h.Me)
}

func (h *APIHandler) RegisterOnboardingRoutes(rg *gin.RouterGroup) {
	rg.POST("/register", h.Onboard)
	rg.POST("/register/google", h.GoogleOnboard)
	rg.GET("/phone-check", h.CheckPhone)
}

// CheckPhone permite al wizard validar (público, sin auth) si un teléfono
// ya está registrado por algún customer activo. No expone qué negocio es.
//
//	GET /api/v1/onboarding/phone-check?phone=+525512345678 → {"exists": bool}
func (h *APIHandler) CheckPhone(c *gin.Context) {
	phone := c.Query("phone")
	if phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone requerido"})
		return
	}
	exists, err := h.service.CheckPhoneExists(phone)
	if err != nil {
		c.Error(apperror.Internal("phone-check failed", err)) //nolint:errcheck
		return
	}
	c.JSON(http.StatusOK, gin.H{"exists": exists})
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

// ForgotPassword always responds 200 (with a neutral message) to avoid
// leaking which emails are registered. Only rate-limit / infra failures
// surface as non-200 via the error middleware.
func (h *APIHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email es requerido"})
		return
	}

	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		c.Error(toAppError(err)) //nolint:errcheck
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Si el email está registrado, enviaremos un enlace para restablecer la contraseña.",
	})
}

// ResetPassword validates the token and sets the new password.
func (h *APIHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token y new_password (min 8) son requeridos"})
		return
	}

	if err := h.service.ResetPassword(c.ClientIP(), req.Token, req.NewPassword); err != nil {
		c.Error(toAppError(err)) //nolint:errcheck
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contraseña actualizada correctamente."})
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

// LinkGoogle vincula una cuenta Google al admin del JWT actual.
func (h *APIHandler) LinkGoogle(c *gin.Context) {
	adminID, ok := currentAdminID(c)
	if !ok {
		c.Error(apperror.BadRequest("token sin admin_id (usar login JWT)", nil)) //nolint:errcheck
		return
	}

	var req LinkGoogleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("google_token es requerido", err)) //nolint:errcheck
		return
	}

	admin, err := h.service.LinkGoogle(adminID, req.GoogleToken)
	if err != nil {
		c.Error(toAppError(err)) //nolint:errcheck
		return
	}
	c.JSON(http.StatusOK, adminToSummary(admin))
}

// UnlinkGoogle remueve la vinculación con Google del admin del JWT actual.
func (h *APIHandler) UnlinkGoogle(c *gin.Context) {
	adminID, ok := currentAdminID(c)
	if !ok {
		c.Error(apperror.BadRequest("token sin admin_id (usar login JWT)", nil)) //nolint:errcheck
		return
	}
	admin, err := h.service.UnlinkGoogle(adminID)
	if err != nil {
		c.Error(toAppError(err)) //nolint:errcheck
		return
	}
	c.JSON(http.StatusOK, adminToSummary(admin))
}

// Me devuelve el admin del JWT actual (incluye estado de vinculación con Google).
func (h *APIHandler) Me(c *gin.Context) {
	adminID, ok := currentAdminID(c)
	if !ok {
		c.Error(apperror.BadRequest("token sin admin_id (usar login JWT)", nil)) //nolint:errcheck
		return
	}
	admin, err := h.service.repo.GetByID(adminID)
	if err != nil {
		c.Error(toAppError(err)) //nolint:errcheck
		return
	}

	resp := MeResponse{AdminSummary: adminToSummary(admin)}
	if h.flags != nil {
		flags, err := h.flags.EnabledFor(c.Request.Context(), admin.CustomerID)
		if err != nil {
			// Flags are best-effort for UI gating; never fail /auth/me on them.
			flags = map[string]bool{}
		}
		resp.Flags = flags
	}
	c.JSON(http.StatusOK, resp)
}

func adminToSummary(a *Admin) AdminSummary {
	return AdminSummary{
		ID:           a.ID,
		Email:        a.Email,
		CustomerID:   a.CustomerID,
		GoogleEmail:  a.GoogleEmail,
		FullName:     a.FullName,
		AvatarURL:    a.AvatarURL,
		Locale:       a.Locale,
		HostedDomain: a.HostedDomain,
	}
}

func currentAdminID(c *gin.Context) (string, bool) {
	v, exists := c.Get("admin_id")
	if !exists {
		return "", false
	}
	id, ok := v.(string)
	if !ok || id == "" {
		return "", false
	}
	return id, true
}

func toAppError(err error) error {
	if _, ok := err.(*apperror.AppError); ok {
		return err
	}
	return apperror.Internal(err.Error(), err)
}
