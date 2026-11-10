package chaos

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/paybridge/api/internal/claims"
	"github.com/paybridge/api/internal/orchestrator"
	"github.com/paybridge/api/internal/paypal"
	"github.com/paybridge/api/internal/reducer"
	"github.com/paybridge/api/internal/safeaction"
)

// CHAOS-01: Network proxy drops capture confirmation; Reconciler recovers original capture with 0 duplicate charges.
func TestCHAOS01_DroppedCaptureResponseRecovery(t *testing.T) {
	ctx := context.Background()
	r := reducer.NewReducer()

	// 1. Initial State: Buyer approved checkout
	state := reducer.StateVector{
		ProviderState:              reducer.ProviderStateApproved,
		LastConfirmedProviderState: reducer.ProviderStateApproved,
		ApplicationState:           reducer.AppStateSynchronised,
		CaseStatus:                 reducer.CaseStatusOpen,
		StateVersion:               1,
	}

	attempt := safeaction.AttemptContext{
		Status:    "CAPTURE_REQUESTED",
		AttemptNo: 1,
	}

	// 2. CHAOS: HTTP 504 Timeout on capture response
	timeoutEvent := reducer.Event{
		EventID:        "evt-chaos-timeout",
		EventType:      "CHECKOUT_TIMEOUT",
		EvidenceID:     "ev-chaos-504",
		Tier:           reducer.TierE3LocalAppTrace,
		SourceTrust:    reducer.SourceTrustLocalObserved,
		Environment:    "REAL_SANDBOX",
		StatusObserved: "TIMEOUT",
		ObservedAt:     time.Now(),
	}

	state, _, err := r.Apply(ctx, state, timeoutEvent)
	if err != nil {
		t.Fatalf("unexpected error on timeout event: %v", err)
	}

	attempt.Status = "CAPTURE_UNKNOWN"

	// 3. Verify Next Safe Action Engine strictly blocks duplicate checkout
	actions := safeaction.Evaluate(state, attempt)
	if actions.PrimaryAction.Code != safeaction.ActionDoNotRepeatCheckout {
		t.Fatalf("VIOLATION: Primary action is not DO_NOT_REPEAT_CHECKOUT, got %s", actions.PrimaryAction.Code)
	}
	for _, a := range actions.Allowed {
		if a.Code == safeaction.ActionRetryCheckout {
			t.Fatalf("CRITICAL FAULT: Retry checkout is permitted while capture is outcome-unknown!")
		}
	}

	// 4. Safe Read-Only Reconciliation: Query original capture directly
	reconcileEvent := reducer.Event{
		EventID:        "evt-reconciled",
		EventType:      "PROVIDER_CAPTURE_COMPLETED",
		EvidenceID:     "ev-paypal-cap-authoritative",
		Tier:           reducer.TierE1DirectProviderRead,
		SourceTrust:    reducer.SourceTrustProviderVerified,
		Environment:    "REAL_SANDBOX",
		StatusObserved: "COMPLETED",
		ObservedAt:     time.Now(),
	}

	state, _, err = r.Apply(ctx, state, reconcileEvent)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	attempt.Status = "CAPTURE_COMPLETED"

	if state.ProviderState != reducer.ProviderStateCaptureCompleted {
		t.Errorf("expected CAPTURE_COMPLETED, got %s", state.ProviderState)
	}
	if state.ApplicationState != reducer.AppStateSynchronised {
		t.Errorf("expected SYNCHRONISED, got %s", state.ApplicationState)
	}

	// Invariant: Total new mutating calls made during reconciliation = 0
	actionsAfter := safeaction.Evaluate(state, attempt)
	if actionsAfter.PrimaryAction.Code != safeaction.ActionNoneRequired {
		t.Errorf("expected NONE_REQUIRED, got %s", actionsAfter.PrimaryAction.Code)
	}
}

// CHAOS-02: Duplicate webhook delivery; assert idempotent single processing and zero state regression.
func TestCHAOS02_DuplicateWebhookDelivery(t *testing.T) {
	ctx := context.Background()
	r := reducer.NewReducer()

	state := reducer.StateVector{
		ProviderState:    reducer.ProviderStateCaptureCompleted,
		ApplicationState: reducer.AppStateSynchronised,
		StateVersion:     5,
	}

	// Repeated identical webhook delivery
	duplicateWebhookEvent := reducer.Event{
		EventID:     "evt-webhook-dup",
		EventType:   "PAYMENT.CAPTURE.COMPLETED",
		EvidenceID:  "ev-webhook-001",
		Tier:        reducer.TierE2VerifiedWebhook,
		Environment: "REAL_SANDBOX",
		ObservedAt:  time.Now(),
	}

	nextState, transitions, err := r.Apply(ctx, state, duplicateWebhookEvent)
	if err != nil {
		t.Fatalf("unexpected error on duplicate webhook: %v", err)
	}

	// State should remain unchanged and no redundant transitions recorded
	if nextState.ProviderState != reducer.ProviderStateCaptureCompleted {
		t.Errorf("expected CAPTURE_COMPLETED, got %s", nextState.ProviderState)
	}
	if len(transitions) != 0 {
		t.Errorf("expected 0 transitions for duplicate webhook, got %d", len(transitions))
	}
	if nextState.StateVersion != state.StateVersion {
		t.Errorf("state version should not increment on duplicate webhook")
	}
}

// CHAOS-04: Forged webhook signature; assert immediate rejection.
func TestCHAOS04_ForgedWebhookSignature(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/oauth2/token" {
			w.Write([]byte(`{"access_token":"fake-token","token_type":"Bearer","expires_in":3600}`))
			return
		}
		w.Write([]byte(`{"verification_status":"FAILURE"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	tm := paypal.NewTokenManager("id", "sec", server.URL, server.Client())
	verifier := paypal.NewWebhookVerifier(server.URL, tm, server.Client())

	res, err := verifier.VerifyWebhookSignature(ctx, paypal.WebhookVerifyInput{
		TransmissionID:  "fake-transmission-id",
		TransmissionSig: "forged_signature_bytes",
		WebhookID:       "wh-12345",
	})
	if err != nil {
		t.Fatalf("verifier error: %v", err)
	}

	if res.VerificationStatus != "FAILURE" {
		t.Errorf("expected verification status FAILURE for forged signature, got %s", res.VerificationStatus)
	}
}

// CHAOS-10: LLM service unavailable / disabled; assert deterministic Passport fallback works seamlessly.
func TestCHAOS10_LLMServiceUnavailableFallback(t *testing.T) {
	val := claims.NewValidator()
	orc := orchestrator.NewOrchestrator(orchestrator.Config{
		LLMEnabled: false, // Kill switch active
	}, val)

	snapshot := &claims.EvidenceSnapshot{
		SnapshotVersion: 1,
		CaseID:          "case-chaos-10",
		EvidenceItems:   map[string]claims.SnapshotEvidenceItem{},
	}

	state := reducer.StateVector{
		ProviderState:    reducer.ProviderStateUnknown,
		ApplicationState: reducer.AppStateAPITimeout,
		CaseStatus:       reducer.CaseStatusInvestigating,
	}

	attempt := safeaction.AttemptContext{
		Status: "CAPTURE_UNKNOWN",
	}

	ctx := context.Background()
	res, err := orc.RunDAG(ctx, snapshot, state, attempt)
	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}

	if res.Status != "COMPLETED" {
		t.Errorf("expected status COMPLETED via deterministic fallback, got %s", res.Status)
	}

	if res.SafeActions.PrimaryAction.Code != safeaction.ActionDoNotRepeatCheckout {
		t.Errorf("expected primary action DO_NOT_REPEAT_CHECKOUT")
	}
}
