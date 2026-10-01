package apis

import (
	"log/slog"
	"net/http"

	"glossias/src/apis/billing"
	"glossias/src/apis/handlers"
	"glossias/src/apis/users"
	"glossias/src/pkg/models"

	"github.com/gorilla/mux"
)

// Handler is a wrapper that delegates to the handlers package
type Handler struct {
	*handlers.Handler
	users   *users.Handler
	billing *billing.Handler
	logger  *slog.Logger
}

// NewHandler creates a new API handler. produceGrading may be nil to run
// without AI grading.
func NewHandler(logger *slog.Logger, produceGrading *models.ProduceGradingService) *Handler {
	var gateway billing.Gateway
	if g, err := billing.NewStripeGatewayFromEnv(); err != nil {
		logger.Warn("Stripe not configured; checkout disabled", "error", err)
	} else {
		gateway = g
		logger.Info("Stripe checkout enabled")
	}
	logger.Info("paywall", "enabled", models.PaywallEnabled())
	return &Handler{
		Handler: handlers.NewHandler(logger, produceGrading),
		users:   users.NewHandler(logger),
		billing: billing.NewHandler(logger, gateway),
		logger:  logger,
	}
}

// RegisterRoutes registers all public story API routes under /api/stories
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Base is /api/stories
	storiesRouter := router.PathPrefix("/stories").Subrouter()
	h.Handler.RegisterRoutes(storiesRouter)
	h.users.RegisterRoutes(router)
	h.billing.RegisterRoutes(router)
}

func (h *Handler) BillingWebhook() http.HandlerFunc {
	return h.billing.HandleWebhook
}
