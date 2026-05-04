package pushcard

import "github.com/gin-gonic/gin"

// APIHandler exposes HTTP endpoints for pushcard. Routes are registered in FID-6.
type APIHandler struct {
	service *Service
}

func NewAPIHandler(service *Service) *APIHandler {
	return &APIHandler{service: service}
}

// RegisterRoutes wires the handlers under the given group. Filled in on FID-6.
func (h *APIHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// no-op for now — endpoints land in FID-6
	_ = rg
}
