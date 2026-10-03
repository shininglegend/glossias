package apis

import (
	"context"
	"log/slog"
	"net/http"
	"os"

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
	g, gatewayErr := billing.NewStripeGatewayFromEnv()
	if gatewayErr != nil {
		logger.Warn("Stripe not configured; checkout disabled", "error", gatewayErr)
	} else {
		gateway = g
		logger.Info("Stripe checkout enabled")
	}
	// With the paywall on, no gateway means nobody can pay: fail open and alert.
	billing.NoteGatewayStartup(context.Background(), gatewayErr)
	logger.Info("paywall", "enabled", models.PaywallEnabled(), "payments_paused", models.PaymentsPaused(),
		"PAYWALL_ENABLED", os.Getenv("PAYWALL_ENABLED"))
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
