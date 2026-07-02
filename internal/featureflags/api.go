package featureflags

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/theluisbolivar/fidel-quick/internal/apperror"
)

// APIHandler exposes the admin feature-flag endpoints.
type APIHandler struct {
	service *Service
}

func NewAPIHandler(service *Service) *APIHandler {
	return &APIHandler{service: service}
}

// RegisterRoutes wires the admin flag endpoints under /admin/flags. Mount under
// the JWT-protected group.
//
//	GET /admin/flags       — list every flag definition.
//	PUT /admin/flags/:key  — create or update a flag (toggle without redeploy).
func (h *APIHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/admin/flags", h.list)
	rg.PUT("/admin/flags/:key", h.update)
}

func (h *APIHandler) list(c *gin.Context) {
	flags, err := h.service.List(c.Request.Context())
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}
	if flags == nil {
		flags = []Flag{}
	}
	c.JSON(http.StatusOK, flags)
}

func (h *APIHandler) update(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.Error(apperror.BadRequest("key requerido", nil)) //nolint:errcheck
		return
	}

	var in UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(apperror.BadRequest("cuerpo inválido", err)) //nolint:errcheck
		return
	}

	flag, err := h.service.Update(c.Request.Context(), key, in)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}
	c.JSON(http.StatusOK, flag)
}
