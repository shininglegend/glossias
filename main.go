package main

import (
	"context"
	"glossias/src/alerting"
	"glossias/src/auth"
	"glossias/src/logging"
	"glossias/src/pkg/database"
	"glossias/src/pkg/models"
	"glossias/src/server"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
)

func main() {
	// Load .env first so LOG_FORMAT / LOG_LEVEL are available when the
	// handler is constructed (JSON in production, pretty on a local TTY).
	dotenvErr := godotenv.Load()

	// The level is a LevelVar so it can be set after the logger exists: the
	// handler reads it on every record.
	var level slog.LevelVar
	level.Set(slog.LevelDebug) // default until LOG_LEVEL is read
	logger := slog.New(logging.NewHandler(os.Stdout, &logging.Options{
		Level:     &level,
		UseColors: true,
	}))

	// Packages without an injected logger (models) log through slog.Default.
	slog.SetDefault(logger)

	if dotenvErr != nil {
		logger.Warn("No .env file found, relying on environment variables")
	}
	// LOG_LEVEL: DEBUG (default), INFO, WARN, ERROR.
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		if err := level.UnmarshalText([]byte(v)); err != nil {
			logger.Warn("invalid LOG_LEVEL, keeping DEBUG", "value", v)
		}
	}

	// DEV_USER disables authentication; refuse to start with it anywhere but
	// a developer's machine.
	if err := auth.CheckDevBypass(os.Getenv("DEV_USER"), os.Getenv("PUBLIC_APP_URL")); err != nil {
		logger.Error("refusing to start", "error", err)
		os.Exit(1)
	}
	if os.Getenv("DEV_USER") != "" {
		logger.Warn("DEV_USER is set: authentication is bypassed for requests with the dev_auth header (local development only)")
	}

	// Initialize database with automatic reconnection support
	// USE_POOL=true uses pgxpool, USE_POOL=false uses database/sql, no DATABASE_URL uses mock
	dbPath := "" // Not used for PostgreSQL
	db, err := database.InitDBWithReconnect(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// Set the DB for the models package
	models.SetDB(db.RawConn())
	// Set the storage client for the models package
	storageUrl := os.Getenv("STORAGE_URL")
	storageKey := os.Getenv("STORAGE_API_KEY")
	if storageUrl == "" || storageKey == "" {
		logger.Warn("STORAGE_URL or STORAGE_API_KEY environment variable not set, storage operations will fail")
	}
	models.SetStorageClient(storageUrl, storageKey)
	// Initialize cache
	if err := models.SetCache(); err != nil {
		logger.Error("Failed to initialize cache", "error", err)
		os.Exit(1)
	}

	// Clerk stuff
	clerk_key := os.Getenv("CLERK_SECRET_KEY")
	if clerk_key == "" {
		logger.Error("CLERK_SECRET_KEY environment variable not set. All auth will fail.")
	}
	clerk.SetKey(clerk_key)

	// AI grading of Produce submissions runs in the background and is optional:
	// without an API key submissions are stored ungraded.
	// The grader's system prompt is versioned in the database and edited on the
	// admin System page; make sure the first version is on record.
	if err := models.EnsureProduceGradingPrompt(context.Background()); err != nil {
		logger.Warn("Could not seed the Produce grading prompt; grading will use the built-in default", "error", err)
	}
	// Operator alerting: conditions that need a human (a payments pause, an
	// AI grading outage) raise Better Stack incidents directly rather than
	// relying on a log line being noticed. One alerter serves every incident.
	if alerter, ok := alerting.NewBetterStackFromEnv(logger); ok {
		models.SetAlerter(alerter)
		logger.Info("incident alerting enabled", "destinations", alerter.Destinations())
	} else {
		logger.Warn("BETTERSTACK_UPTIME_TOKEN/BETTERSTACK_REQUESTER_EMAIL and BETTERSTACK_WEBHOOK_URL not set; a payments pause or an AI grading outage will only appear in logs")
	}
	var produceGrading *models.ProduceGradingService
	if grader, ok := models.NewAnthropicGraderFromEnv(); ok {
		produceGrading = models.NewProduceGradingService(grader, logger)
		logger.Info("AI grading enabled", "model", models.GradingModel)
	} else {
		logger.Warn("ANTHROPIC_API_KEY not set; Produce submissions will not be AI-graded")
	}
	// Payments fail-open: re-apply a pause persisted before this start so a
	// redeploy cannot hide a lifted paywall; the alerter is told again.
	if err := models.LoadPaymentsState(context.Background()); err != nil {
		logger.Error("could not read persisted payments state; assuming not paused", "error", err)
	}

	// Clerk: require Authorization: Bearer <token> on every request (unless dev auth bypass)
	var apiAuth mux.MiddlewareFunc
	if os.Getenv("DEV_USER") == "" {
		authorizedParty := os.Getenv("AUTHORIZED_PARTY")
		if authorizedParty == "" {
			logger.Warn("AUTHORIZED_PARTY environment variable not set")
			// It's not actually needed, but can cause problems if missing.
			apiAuth = clerkhttp.RequireHeaderAuthorization()
		} else {
			apiAuth = clerkhttp.RequireHeaderAuthorization(
				clerkhttp.AuthorizedPartyMatches(authorizedParty),
			)
		}
	}

	r := server.NewRouter(server.Config{
		Logger:         logger,
		ProduceGrading: produceGrading,
		RateLimit:      auth.RateLimitMiddleware(logger),
		Auth:           auth.Middleware(logger),
		APIAuth:        apiAuth,
	})

	// Select correct port and start the server
	port := os.Getenv("PORT")
	if port == "" {
		logger.Error("PORT environment variable not set")
		os.Exit(1)
	}

	srv := &http.Server{
		Handler:      r,
		Addr:         ":" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	logger.Info("starting server", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
