package landing

import (
	"database/sql"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/theluisbolivar/fidel-quick/internal/deeplink"
	"github.com/theluisbolivar/fidel-quick/internal/resolver"
)

type Handler struct {
	repo         resolver.Repository
	log          *slog.Logger
	displayPhone string
	templates    *template.Template
}

type businessData struct {
	ID             string
	Name           string
	Slug           string
	LogoURL        string
	Description    string
	WelcomeMessage string
	DeeplinkURL    string
}

func NewHandler(repo resolver.Repository, log *slog.Logger, displayPhone string) *Handler {
	tmpl := template.Must(template.ParseGlob("internal/landing/templates/*.html"))
	return &Handler{
		repo:         repo,
		log:          log,
		displayPhone: displayPhone,
		templates:    tmpl,
	}
}

// Join handles GET /unirse/:slug — renders the landing page for a business.
func (h *Handler) Join(c *gin.Context) {
	slug := c.Param("slug")

	id, name, slugOut, logoURL, desc, welcome, err := h.repo.GetCustomerBySlug(c.Request.Context(), slug)
	if err != nil {
		if err == sql.ErrNoRows {
			c.Status(http.StatusNotFound)
			h.templates.ExecuteTemplate(c.Writer, "404.html", nil)
			return
		}
		h.log.Error("failed to query customer", "slug", slug, "error", err)
		c.String(http.StatusInternalServerError, "Error interno")
		return
	}

	b := businessData{
		ID:             id,
		Name:           name,
		Slug:           slugOut,
		LogoURL:        logoURL,
		Description:    desc,
		WelcomeMessage: welcome,
		DeeplinkURL:    deeplink.WhatsAppURL(h.displayPhone, id, name),
	}

	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(c.Writer, "join.html", b); err != nil {
		h.log.Error("failed to render template", "error", err)
	}
}
