package claims

import (
	"errors"
	"fmt"
	"strings"

	"github.com/paybridge/api/internal/reducer"
)

var (
	ErrInvalidClaimSchema  = errors.New("claim failed JSON schema validation")
	ErrMissingCitation     = errors.New("confirmed claim lacks required evidence citation")
	ErrEvidenceNotFound    = errors.New("cited evidence ID does not exist in frozen snapshot")
	ErrInsufficientTier    = errors.New("confirmed claim requires Tier E1 or E2 evidence")
	ErrForbiddenTerminology = errors.New("claim contains forbidden misleading terminology")
)

// ClaimKind represents epistemic certainty
type ClaimKind string

const (
	ClaimKindConfirmed ClaimKind = "confirmed"
	ClaimKindInference ClaimKind = "inference"
	ClaimKindUnknown   ClaimKind = "unknown"
)

// Claim represents a structured finding produced by an AI specialist agent
type Claim struct {
	Code              string    `json:"code"`
	Kind              ClaimKind `json:"kind"`
	Statement         string    `json:"statement"`
	EvidenceIDs       []string  `json:"evidence_ids"`
	InferenceRuleID   string    `json:"inference_rule_id,omitempty"`
	ViewerScope       string    `json:"viewer_scope"` // SENDER, RECEIVER, OPS, ALL
	Validated         bool      `json:"validated"`
	DowngradeReason   string    `json:"downgrade_reason,omitempty"`
}

// SnapshotEvidenceItem represents an evidence item in the frozen snapshot
type SnapshotEvidenceItem struct {
	ID          string               `json:"id"`
	Tier        reducer.EvidenceTier `json:"tier"`
	SourceTrust reducer.SourceTrust  `json:"source_trust"`
	Kind        string               `json:"kind"`
	Verified    bool                 `json:"verified"`
}

// EvidenceSnapshot represents the immutable state against which claims are verified
type EvidenceSnapshot struct {
	SnapshotVersion int64                            `json:"snapshot_version"`
	CaseID          string                           `json:"case_id"`
	EvidenceItems   map[string]SnapshotEvidenceItem `json:"evidence_items"`
}

// Validator validates claims against evidence and security invariants
type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

// ValidateClaim validates and enforces citation integrity, tier rules, and calm copy rules.
// Downgrades ungrounded confirmed claims to 'unknown' rather than propagating hallucinated claims.
func (v *Validator) ValidateClaim(claim Claim, snapshot *EvidenceSnapshot) (Claim, error) {
	out := claim

	// Schema basic sanity check
	if strings.TrimSpace(out.Code) == "" || strings.TrimSpace(out.Statement) == "" {
		return out, ErrInvalidClaimSchema
	}

	// Forbidden language check: Never allow "failed" to describe an unknown state
	statementLower := strings.ToLower(out.Statement)
	if strings.Contains(statementLower, "payment failed") || strings.Contains(statementLower, "transaction failed") {
		return out, fmt.Errorf("%w: cannot use 'payment failed' on ambiguous payment outcomes", ErrForbiddenTerminology)
	}

	if out.Kind == ClaimKindConfirmed {
		// Citation Gate: Every confirmed claim MUST cite at least one evidence ID
		if len(out.EvidenceIDs) == 0 {
			out.Kind = ClaimKindUnknown
			out.DowngradeReason = "Downgraded to unknown: missing required evidence citation"
			out.Validated = false
			return out, ErrMissingCitation
		}

		// Verify that all cited evidence IDs exist in snapshot and check tiers
		hasE1orE2 := false
		for _, eid := range out.EvidenceIDs {
			item, exists := snapshot.EvidenceItems[eid]
			if !exists {
				out.Kind = ClaimKindUnknown
				out.DowngradeReason = fmt.Sprintf("Downgraded to unknown: cited evidence ID %s does not exist in snapshot", eid)
				out.Validated = false
				return out, ErrEvidenceNotFound
			}

			// Tier Gate: Confirmed claim requires Tier E1 (direct GET) or Tier E2 (verified webhook)
			if item.Tier == reducer.TierE1DirectProviderRead || item.Tier == reducer.TierE2VerifiedWebhook {
				if item.Verified {
					hasE1orE2 = true
				}
			}
		}

		if !hasE1orE2 {
			out.Kind = ClaimKindInference
			out.DowngradeReason = "Downgraded to inference: confirmed claims require verified Tier E1 or E2 evidence"
			out.Validated = false
			return out, ErrInsufficientTier
		}
	}

	out.Validated = true
	return out, nil
}
