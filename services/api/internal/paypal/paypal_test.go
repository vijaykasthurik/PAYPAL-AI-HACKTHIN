package paypal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestReaderHasNoMutatingMethods(t *testing.T) {
	// Architectural reflection test: Reader interface MUST NOT contain mutating verbs
	readerType := reflect.TypeOf((*Reader)(nil)).Elem()
	for i := 0; i < readerType.NumMethod(); i++ {
		methodName := readerType.Method(i).Name
		lower := strings.ToLower(methodName)
		if strings.HasPrefix(lower, "create") ||
			strings.HasPrefix(lower, "post") ||
			strings.HasPrefix(lower, "capture") ||
			strings.HasPrefix(lower, "refund") ||
			strings.HasPrefix(lower, "delete") ||
			strings.HasPrefix(lower, "update") {
			t.Errorf("Reader interface violates GET-only invariant! Found mutating method: %s", methodName)
		}
	}
}

func TestPayPalAdapter_FakeServerFlows(t *testing.T) {
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/oauth2/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "fake-access-token-12345",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})

		case "/v2/checkout/orders":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			reqID := r.Header.Get("PayPal-Request-Id")
			if reqID == "" {
				http.Error(w, "missing request id", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":     "ORD-TEST-123",
				"status": "CREATED",
				"intent": "CAPTURE",
			})

		case "/v2/checkout/orders/ORD-TEST-123":
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":     "ORD-TEST-123",
				"status": "APPROVED",
			})

		case "/v2/checkout/orders/ORD-TEST-123/capture":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			// Simulate 504 Gateway Timeout when header flag is present
			if r.Header.Get("X-Simulate-Timeout") == "true" {
				w.WriteHeader(http.StatusGatewayTimeout)
				w.Write([]byte(`{"message":"Gateway Timeout"}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":     "CAP-TEST-789",
				"status": "COMPLETED",
			})

		case "/v2/payments/captures/CAP-TEST-789":
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":     "CAP-TEST-789",
				"status": "COMPLETED",
			})

		case "/v1/notifications/verify-webhook-signature":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"verification_status": "SUCCESS",
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer fakeServer.Close()

	ctx := context.Background()
	tm := NewTokenManager("fake_client", "fake_secret", fakeServer.URL, fakeServer.Client())
	reader := NewReader(fakeServer.URL, tm, fakeServer.Client())
	mutator := NewMutator(fakeServer.URL, tm, fakeServer.Client())
	verifier := NewWebhookVerifier(fakeServer.URL, tm, fakeServer.Client())

	// 1. Mutator: Create Order
	ord, evO, err := mutator.CreateOrder(ctx, CreateOrderInput{
		RequestID: "req-create-1",
		Amount:    "100.00",
		Currency:  "USD",
		InvoiceID: "INV-001",
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}
	if ord.ID != "ORD-TEST-123" || ord.Status != "CREATED" {
		t.Errorf("unexpected order response: %+v", ord)
	}
	if evO.PayloadSHA256 == "" {
		t.Errorf("expected evidence hash populated")
	}

	// 2. Reader: Show Order
	showOrd, _, err := reader.ShowOrder(ctx, EnvSandbox, "ORD-TEST-123")
	if err != nil {
		t.Fatalf("ShowOrder failed: %v", err)
	}
	if showOrd.Status != "APPROVED" {
		t.Errorf("expected APPROVED, got %s", showOrd.Status)
	}

	// 3. Mutator: Capture Order
	capRes, evC, err := mutator.CaptureOrder(ctx, CaptureInput{
		RequestID: "req-cap-1",
		OrderID:   "ORD-TEST-123",
	})
	if err != nil {
		t.Fatalf("CaptureOrder failed: %v", err)
	}
	if capRes.ID != "CAP-TEST-789" || capRes.Status != "COMPLETED" {
		t.Errorf("unexpected capture: %+v", capRes)
	}
	if evC.PayloadSHA256 == "" {
		t.Errorf("expected capture evidence hash populated")
	}

	// 4. Reader: Show Capture
	showCap, _, err := reader.ShowCapture(ctx, EnvSandbox, "CAP-TEST-789")
	if err != nil {
		t.Fatalf("ShowCapture failed: %v", err)
	}
	if showCap.Status != "COMPLETED" {
		t.Errorf("expected COMPLETED, got %s", showCap.Status)
	}

	// 5. Verifier: Verify Webhook
	verRes, err := verifier.VerifyWebhookSignature(ctx, WebhookVerifyInput{
		TransmissionID: "trans-1",
		WebhookID:      "hook-1",
	})
	if err != nil {
		t.Fatalf("VerifyWebhookSignature failed: %v", err)
	}
	if verRes.VerificationStatus != "SUCCESS" {
		t.Errorf("expected SUCCESS, got %s", verRes.VerificationStatus)
	}
}

func TestClassifyResponse(t *testing.T) {
	// 504 Timeout -> unknown_missing_response
	class := ClassifyResponse(http.StatusGatewayTimeout, map[string]any{}, nil)
	if class != ClassUnknownMissingResponse {
		t.Errorf("expected unknown_missing_response for 504, got %s", class)
	}

	// Context Deadline -> unknown_missing_response
	class = ClassifyResponse(0, nil, context.DeadlineExceeded)
	if class != ClassUnknownMissingResponse {
		t.Errorf("expected unknown_missing_response for DeadlineExceeded, got %s", class)
	}

	// 200 with COMPLETED -> success
	class = ClassifyResponse(http.StatusOK, map[string]any{"status": "COMPLETED"}, nil)
	if class != ClassSuccess {
		t.Errorf("expected success for 200 COMPLETED, got %s", class)
	}

	// 422 with INSTRUMENT_DECLINED -> explicit_decline
	class = ClassifyResponse(http.StatusUnprocessableEntity, map[string]any{"name": "INSTRUMENT_DECLINED"}, nil)
	if class != ClassExplicitDecline {
		t.Errorf("expected explicit_decline for INSTRUMENT_DECLINED, got %s", class)
	}
}
