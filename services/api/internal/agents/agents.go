package agents

import (
	"fmt"
	"strings"

	"github.com/paybridge/api/internal/claims"
	"github.com/paybridge/api/internal/safeaction"
)

type AgentID string

const (
	AgentA01PaymentReadiness  AgentID = "A01"
	AgentA02TrafficService    AgentID = "A02"
	AgentA03SenderAnalysis    AgentID = "A03"
	AgentA04ReceiverAnalysis  AgentID = "A04"
	AgentA05JourneyTracker    AgentID = "A05"
	AgentA06RootCauseAnalyst  AgentID = "A06"
	AgentA07CurrencyIntel     AgentID = "A07"
	AgentA08RecoveryPlanner   AgentID = "A08"
	AgentA09DualCommunicator  AgentID = "A09"
	AgentA10ResolutionTracker AgentID = "A10"
)

// AgentOutput models standard structured output from specialist agents
type AgentOutput struct {
	AgentID     AgentID        `json:"agent_id"`
	AgentName   string         `json:"agent_name"`
	Summary     string         `json:"summary"`
	Claims      []claims.Claim `json:"claims"`
	NextActions []safeaction.Action `json:"next_actions,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// WrapUntrustedText applies prompt injection defense delimiters
func WrapUntrustedText(label, text string) string {
	sanitized := strings.ReplaceAll(text, "</untrusted_data>", "")
	return fmt.Sprintf("<untrusted_data label=%q>\n%s\n</untrusted_data>", label, sanitized)
}

// GenerateDeterministicOutput produces reliable rule-based output when LLM_ENABLED=false (Demo E)
func GenerateDeterministicOutput(agentID AgentID, contextData map[string]any) AgentOutput {
	switch agentID {
	case AgentA05JourneyTracker:
		return AgentOutput{
			AgentID:   AgentA05JourneyTracker,
			AgentName: "Journey Tracker",
			Summary:   "Chronological lifecycle timeline reconstructed from local app traces and provider status updates.",
			Claims: []claims.Claim{
				{
					Code:        "JOURNEY_TIMEOUT_DETECTED",
					Kind:        claims.ClaimKindConfirmed,
					Statement:   "Network gateway timeout detected during PayPal capture call. The response was not received before deadline.",
					EvidenceIDs: []string{fmt.Sprintf("%v", contextData["timeout_evidence_id"])},
					ViewerScope: "ALL",
				},
			},
		}

	case AgentA03SenderAnalysis:
		return AgentOutput{
			AgentID:   AgentA03SenderAnalysis,
			AgentName: "Sender Analysis",
			Summary:   "Sender checkout attempt initiated with verified authorization.",
			Claims: []claims.Claim{
				{
					Code:        "SENDER_CAPTURE_IN_FLIGHT",
					Kind:        claims.ClaimKindConfirmed,
					Statement:   "The sender submitted capture request with stable idempotency key.",
					EvidenceIDs: []string{fmt.Sprintf("%v", contextData["order_evidence_id"])},
					ViewerScope: "SENDER",
				},
			},
		}

	case AgentA06RootCauseAnalyst:
		return AgentOutput{
			AgentID:   AgentA06RootCauseAnalyst,
			AgentName: "Root Cause Analyst",
			Summary:   "Primary cause: Network communication drop between PayBridge client and PayPal gateway during capture execution (F02 Transient Network Drop).",
			Claims: []claims.Claim{
				{
					Code:        "ROOT_CAUSE_TRANSIENT_DROP",
					Kind:        claims.ClaimKindInference,
					Statement:   "Client experienced HTTP 504 Gateway Timeout while PayPal order status remains pending confirmation.",
					EvidenceIDs: []string{fmt.Sprintf("%v", contextData["timeout_evidence_id"])},
					InferenceRuleID: "RULE_F02_TIMEOUT_DISCONNECT",
					ViewerScope: "ALL",
				},
			},
		}

	case AgentA08RecoveryPlanner:
		return AgentOutput{
			AgentID:   AgentA08RecoveryPlanner,
			AgentName: "Recovery Planner",
			Summary:   "Strictly blocking blind retry checkout. Safe recovery initiated via read-only GET query on original order.",
			NextActions: []safeaction.Action{
				{
					Code:             safeaction.ActionDoNotRepeatCheckout,
					Label:            "Do Not Repeat Checkout",
					Rationale:        "Prevent duplicate payment capture.",
					PreconditionsMet: true,
					DesignatedActor:  "SENDER",
				},
				{
					Code:             safeaction.ActionAwaitProviderConfirmation,
					Label:            "Reconcile Original Order (Read-Only)",
					Rationale:        "Reconciler polling original capture reference.",
					PreconditionsMet: true,
					DesignatedActor:  "SYSTEM",
				},
			},
		}

	case AgentA09DualCommunicator:
		return AgentOutput{
			AgentID:   AgentA09DualCommunicator,
			AgentName: "Dual-Side Communicator",
			Summary:   "Role-specific calm communications formulated without misleading 'failed' terminology.",
			Claims: []claims.Claim{
				{
					Code:        "COMM_SENDER_NOTICE",
					Kind:        claims.ClaimKindConfirmed,
					Statement:   "Your payment is currently being verified with PayPal. Please do not submit another payment while verification is underway.",
					EvidenceIDs: []string{fmt.Sprintf("%v", contextData["order_evidence_id"])},
					ViewerScope: "SENDER",
				},
				{
					Code:        "COMM_RECEIVER_NOTICE",
					Kind:        claims.ClaimKindConfirmed,
					Statement:   "An incoming transfer is undergoing provider confirmation. Updates will reflect once PayPal confirms completion.",
					EvidenceIDs: []string{fmt.Sprintf("%v", contextData["order_evidence_id"])},
					ViewerScope: "RECEIVER",
				},
			},
		}

	default:
		return AgentOutput{
			AgentID:   agentID,
			AgentName: string(agentID),
			Summary:   "Deterministic analysis complete.",
		}
	}
}
