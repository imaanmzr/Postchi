package publicconfig

import (
	"net/http"

	"github.com/imaanmzr/postchi/backend/internal/shared/config"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

type Handler struct {
	cfg *config.Config
}

func NewHandler(cfg *config.Config) *Handler {
	return &Handler{cfg: cfg}
}

type PublicConfig struct {
	SMTPConfigured             bool     `json:"smtp_configured"`
	RegistrationAllowedDomains []string `json:"registration_allowed_domains"`
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	respond.JSON(w, http.StatusOK, PublicConfig{
		SMTPConfigured:             h.cfg.SMTPConfigured(),
		RegistrationAllowedDomains: h.cfg.RegistrationAllowedDomains,
	})
}
