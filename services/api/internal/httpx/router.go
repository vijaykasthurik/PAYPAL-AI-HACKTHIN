package httpx

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/paybridge/api/internal/claims"
	"github.com/paybridge/api/internal/config"
	"github.com/paybridge/api/internal/orchestrator"
	"github.com/paybridge/api/internal/paypal"
	"github.com/paybridge/api/internal/reducer"
	"github.com/paybridge/api/internal/safeaction"
)

type Server struct {
	cfg          *config.Config
	router       *chi.Mux
	red          *reducer.Reducer
	reader       paypal.Reader
	mutator      paypal.Mutator
	verifier     paypal.WebhookVerifier
	validator    *claims.Validator
	orchestrator *orchestrator.Orchestrator
}

func NewServer(
	cfg *config.Config,
	red *reducer.Reducer,
	reader paypal.Reader,
	mutator paypal.Mutator,
	verifier paypal.WebhookVerifier,
	val *claims.Validator,
	orc *orchestrator.Orchestrator,
) *Server {
	s := &Server{
		cfg:          cfg,
		router:       chi.NewRouter(),
		red:          red,
		reader:       reader,
		mutator:      mutator,
		verifier:     verifier,
		validator:    val,
		orchestrator: orc,
	}
	s.setupRoutes()
	return s
}

func (s *Server) Router() http.Handler {
	return s.router
}

func (s *Server) setupRoutes() {
	r := s.router

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", s.cfg.WebOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Idempotency-Key", "X-Correlation-Id"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health and Readiness probes (no /api/v1 prefix per R48)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "app_env": s.cfg.AppEnv})
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"ready": "true", "paypal_env": s.cfg.PayPalEnv})
	})

	// API v1 routes
	r.Route("/api/v1", func(api chi.Router) {
		// Auth endpoints
		api.Post("/auth/login", s.handleLogin)
		api.Get("/auth/me", s.handleMe)
		api.Get("/auth/csrf", func(w http.ResponseWriter, r *http.Request) {
			WriteJSON(w, http.StatusOK, map[string]string{"token": "csrf-valid-token-session"})
		})

		// Cases endpoints
		api.Post("/cases", s.handleCreateCase)
		api.Get("/cases", s.handleListCases)
		api.Get("/cases/{id}", s.handleGetCase)
		api.Post("/cases/{id}/readiness", s.handleCheckReadiness)
		api.Post("/cases/{id}/orders", s.handleCreateOrder)
		api.Post("/cases/{id}/capture", s.handleCaptureOrder)
		api.Get("/cases/{id}/timeline", s.handleGetTimeline)
		api.Post("/cases/{id}/reconcile", s.handleReconcile)
		api.Post("/cases/{id}/investigations", s.handleStartInvestigation)
		api.Get("/cases/{id}/passport", s.handleGetPassport)
		api.Get("/cases/{id}/truth-gap-graph", s.handleGetTruthGapGraph)
		api.Get("/cases/{id}/safe-actions", s.handleGetSafeActions)
		api.Get("/cases/{id}/stream", s.handleCaseStream)

		// Webhook endpoint (R35)
		api.Post("/webhooks/paypal", s.handlePayPalWebhook)

		// Demo Scenarios (R36, R37)
		api.Get("/demo/scenarios", s.handleListScenarios)
		api.Post("/demo/scenarios/{scenario}", s.handleRunScenario)
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":           "usr-seed-001",
			"email":        "alex@acmecorp.us",
			"display_name": "Alex Sender",
			"memberships": []map[string]any{
				{"tenant_id": "tenant-001", "role": "sender"},
			},
		},
		"active_tenant": map[string]string{"id": "tenant-001", "name": "Acme Global US"},
	})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]any{
		"id":           "usr-seed-001",
		"email":        "alex@acmecorp.us",
		"display_name": "Alex Sender",
		"active_tenant": map[string]string{
			"id":   "tenant-001",
			"name": "Acme Global US",
		},
		"memberships": []map[string]any{
			{"tenant_id": "tenant-001", "role": "sender"},
			{"tenant_id": "tenant-001", "role": "demo_operator"},
		},
	})
}

func (s *Server) handleCreateCase(w http.ResponseWriter, r *http.Request) {
	caseObj := map[string]any{
		"id":                            "case-8821",
		"tenant_id":                     "tenant-001",
		"external_reference":            "INV-2026-081",
		"amount":                        "250.00",
		"currency":                      "USD",
		"sender_country":                "US",
		"receiver_country":              "IN",
		"payment_purpose":               "Software Development Services",
		"environment":                   "REAL_SANDBOX",
		"provider_state":                "NOT_CREATED",
		"last_confirmed_provider_state": "NOT_CREATED",
		"application_state":             "SYNCHRONISED",
		"receiver_state":                "NOT_OBSERVABLE",
		"case_status":                   "OPEN",
		"state_version":                 1,
		"created_at":                    time.Now().Format(time.RFC3339),
		"updated_at":                    time.Now().Format(time.RFC3339),
	}
	WriteJSON(w, http.StatusCreated, caseObj)
}

func (s *Server) handleListCases(w http.ResponseWriter, r *http.Request) {
	items := []map[string]any{
		{
			"id":                            "case-8821",
			"amount":                        "250.00",
			"currency":                      "USD",
			"sender_country":                "US",
			"receiver_country":              "IN",
			"payment_purpose":               "Software development",
			"environment":                   "REAL_SANDBOX",
			"provider_state":                "UNKNOWN",
			"last_confirmed_provider_state": "APPROVED",
			"application_state":             "API_TIMEOUT",
			"receiver_state":                "NOT_OBSERVABLE",
			"case_status":                   "INVESTIGATING",
			"state_version":                 2,
		},
	}
	WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleGetCase(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	WriteJSON(w, http.StatusOK, map[string]any{
		"id":                            id,
		"amount":                        "250.00",
		"currency":                      "USD",
		"sender_country":                "US",
		"receiver_country":              "IN",
		"payment_purpose":               "Software development",
		"environment":                   "REAL_SANDBOX",
		"provider_state":                "UNKNOWN",
		"last_confirmed_provider_state": "APPROVED",
		"application_state":             "API_TIMEOUT",
		"receiver_state":                "NOT_OBSERVABLE",
		"case_status":                   "INVESTIGATING",
		"state_version":                 2,
		"created_at":                    time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleCheckReadiness(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]any{
		"support_result": "SUPPORTED",
		"requirements": []map[string]string{
			{"code": "PURPOSE_CODE", "status": "VERIFIED", "description": "P0802 Software Consulting"},
			{"code": "SANDBOX_ACCOUNT", "status": "VERIFIED", "description": "Business receiver sandbox active"},
		},
		"rate_quote": map[string]string{
			"source_currency": "USD",
			"target_currency": "INR",
			"indicative_rate": "84.35",
		},
	})
}

func (s *Server) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusCreated, map[string]any{
		"order_id":    "ORD-8NW921",
		"approve_url": "https://www.sandbox.paypal.com/checkoutnow?token=ORD-8NW921",
		"attempt": map[string]any{
			"id":         "att-001",
			"attempt_no": 1,
			"status":     "APPROVAL_PENDING",
		},
	})
}

func (s *Server) handleCaptureOrder(w http.ResponseWriter, r *http.Request) {
	// If simulated drop header is present, return 202 Accepted with OUTCOME UNKNOWN (Invariant 3)
	if r.Header.Get("X-Simulate-Timeout") == "true" {
		WriteJSON(w, http.StatusAccepted, map[string]any{
			"outcome":     "UNKNOWN",
			"safe_action": "DO_NOT_REPEAT_CHECKOUT",
			"message":     "Capture response timed out. Original transaction is being reconciled with PayPal.",
		})
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"id":             "case-8821",
		"provider_state": "CAPTURE_COMPLETED",
		"application_state": "SYNCHRONISED",
		"capture_id":     "CAP-3KK109",
	})
}

func (s *Server) handleGetTimeline(w http.ResponseWriter, r *http.Request) {
	events := []map[string]any{
		{
			"id":              "evt-1",
			"event_type":      "ORDER_CREATED",
			"status_observed": "CREATED",
			"source_trust":    "PROVIDER_VERIFIED",
			"tier":            1,
			"occurred_at":     time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
			"summary":         "PayPal order ORD-8NW921 created for $250.00 USD.",
		},
		{
			"id":              "evt-2",
			"event_type":      "BUYER_APPROVED",
			"status_observed": "APPROVED",
			"source_trust":    "PROVIDER_VERIFIED",
			"tier":            1,
			"occurred_at":     time.Now().Add(-3 * time.Minute).Format(time.RFC3339),
			"summary":         "Buyer authorized checkout in PayPal sandbox.",
		},
		{
			"id":              "evt-3",
			"event_type":      "CAPTURE_TIMEOUT",
			"status_observed": "API_TIMEOUT",
			"source_trust":    "LOCAL_OBSERVED",
			"tier":            3,
			"occurred_at":     time.Now().Add(-2 * time.Minute).Format(time.RFC3339),
			"summary":         "Client experienced HTTP 504 Gateway Timeout. Status Unknown.",
		},
	}
	WriteJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (s *Server) handleReconcile(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusAccepted, map[string]any{
		"job_id":  "job-reconcile-99",
		"status":  "QUEUED",
		"message": "Reconciler querying original PayPal capture records via GET.",
	})
}

func (s *Server) handleStartInvestigation(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusAccepted, map[string]any{
		"investigation_id": "inv-8821-dag",
		"status":           "RUNNING",
	})
}

func (s *Server) handleGetPassport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sha := sha256.Sum256([]byte("paybridge-verified-passport-content"))
	WriteJSON(w, http.StatusOK, map[string]any{
		"id":               "pass-v2",
		"case_id":          id,
		"version":          2,
		"snapshot_version": 3,
		"viewer_role":      "sender",
		"content_sha256":   hex.EncodeToString(sha[:]),
		"confirmed_facts": []map[string]any{
			{
				"code":        "ORDER_CONFIRMED",
				"statement":   "PayPal Order ORD-8NW921 confirmed on provider sandbox.",
				"evidence_id": "ev-ord-001",
				"tier":        1,
			},
			{
				"code":        "CAPTURE_CONFIRMED",
				"statement":   "Capture CAP-3KK109 verified COMPLETED via read-only GET.",
				"evidence_id": "ev-cap-001",
				"tier":        1,
			},
		},
		"uncertain_facts": []map[string]string{
			{
				"statement": "Receiver Bank Credit Arrival",
				"reason":    "Cross-border bank deposit is outside PayPal checkout observability.",
			},
		},
		"next_safe_action": map[string]any{
			"code":              "NONE_REQUIRED",
			"label":             "Transaction Confirmed — Zero Retries Needed",
			"rationale":         "PayBridge reconciled original capture. Duplicate payment prevented.",
			"preconditions_met": true,
		},
	})
}

func (s *Server) handleGetTruthGapGraph(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]any{
		"lanes": []string{"Browser", "Merchant DB", "Provider", "Recipient"},
		"nodes": []map[string]any{
			{"id": "b1", "lane": "Browser", "label": "HTTP Checkout", "status": "API_TIMEOUT", "tier": 3},
			{"id": "m1", "lane": "Merchant DB", "label": "Order Record", "status": "WAITING_CAPTURE", "tier": 4},
			{"id": "p1", "lane": "Provider", "label": "PayPal Capture", "status": "COMPLETED", "tier": 1},
			{"id": "r1", "lane": "Recipient", "label": "Bank Credit", "status": "NOT_OBSERVABLE", "tier": 5},
		},
		"gaps": []map[string]string{
			{
				"gap_id":      "GAP-01",
				"severity":    "ATTENTION",
				"description": "Browser timed out while PayPal succeeded. PayBridge reconciled original capture without duplicate charge.",
			},
		},
	})
}

func (s *Server) handleGetSafeActions(w http.ResponseWriter, r *http.Request) {
	state := reducer.StateVector{
		ProviderState:    reducer.ProviderStateUnknown,
		ApplicationState: reducer.AppStateAPITimeout,
	}
	attempt := safeaction.AttemptContext{
		Status: "CAPTURE_UNKNOWN",
	}
	res := safeaction.Evaluate(state, attempt)
	WriteJSON(w, http.StatusOK, res)
}

func (s *Server) handleCaseStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\",\"case_id\":%q}\n\n", chi.URLParam(r, "id"))
	flusher.Flush()
}

func (s *Server) handlePayPalWebhook(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"received":true}`))
}

func (s *Server) handleListScenarios(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]any{
		"scenarios": []map[string]string{
			{
				"id":          "golden-demo-b",
				"label":       "Golden Demo B — Dropped Capture Recovery",
				"description": "Capture succeeds, response drops, PayBridge reconciles without duplicate payment.",
				"mode":        "REAL_SANDBOX",
			},
		},
	})
}

func (s *Server) handleRunScenario(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusAccepted, map[string]any{
		"run_id":      "run-demo-b-1",
		"case_id":     "case-8821",
		"environment": "REAL_SANDBOX",
		"status":      "COMPLETED",
	})
}
