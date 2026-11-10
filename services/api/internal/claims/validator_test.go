package claims

import (
	"testing"

	"github.com/paybridge/api/internal/reducer"
)

func TestValidator_ConfirmedClaimWithValidE1Evidence(t *testing.T) {
	v := NewValidator()
	snapshot := &EvidenceSnapshot{
		SnapshotVersion: 1,
		CaseID:          "case-123",
		EvidenceItems: map[string]SnapshotEvidenceItem{
			"ev-paypal-get": {
				ID:          "ev-paypal-get",
				Tier:        reducer.TierE1DirectProviderRead,
				SourceTrust: reducer.SourceTrustProviderVerified,
				Verified:    true,
			},
		},
	}

	claim := Claim{
		Code:        "CLAIM_CAPTURE_SUCCESS",
		Kind:        ClaimKindConfirmed,
		Statement:   "PayPal confirmed capture was completed successfully.",
		EvidenceIDs: []string{"ev-paypal-get"},
		ViewerScope: "ALL",
	}

	valid, err := v.ValidateClaim(claim, snapshot)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if !valid.Validated {
		t.Errorf("expected claim to be marked validated")
	}
	if valid.Kind != ClaimKindConfirmed {
		t.Errorf("expected claim to remain confirmed, got %s", valid.Kind)
	}
}

func TestValidator_ForgedCitationRejected(t *testing.T) {
	v := NewValidator()
	snapshot := &EvidenceSnapshot{
		SnapshotVersion: 1,
		CaseID:          "case-123",
		EvidenceItems:   map[string]SnapshotEvidenceItem{},
	}

	claim := Claim{
		Code:        "CLAIM_CAPTURE_SUCCESS",
		Kind:        ClaimKindConfirmed,
		Statement:   "PayPal confirmed capture was completed successfully.",
		EvidenceIDs: []string{"ev-forged-nonexistent"},
		ViewerScope: "ALL",
	}

	valid, err := v.ValidateClaim(claim, snapshot)
	if err == nil {
		t.Fatalf("expected error for forged/missing citation, got nil")
	}
	if valid.Validated {
		t.Errorf("claim should not be marked validated")
	}
	if valid.Kind != ClaimKindUnknown {
		t.Errorf("expected claim kind downgraded to unknown, got %s", valid.Kind)
	}
}

func TestValidator_InsufficientTierDowngraded(t *testing.T) {
	v := NewValidator()
	snapshot := &EvidenceSnapshot{
		SnapshotVersion: 1,
		CaseID:          "case-123",
		EvidenceItems: map[string]SnapshotEvidenceItem{
			"ev-user-say": {
				ID:          "ev-user-say",
				Tier:        reducer.TierE5UserReport,
				SourceTrust: reducer.SourceTrustUserReported,
				Verified:    false,
			},
		},
	}

	claim := Claim{
		Code:        "CLAIM_USER_STATEMENT",
		Kind:        ClaimKindConfirmed, // Claims confirmed but backed only by E5 User Report
		Statement:   "The payer reports their bank was debited.",
		EvidenceIDs: []string{"ev-user-say"},
		ViewerScope: "ALL",
	}

	valid, err := v.ValidateClaim(claim, snapshot)
	if err != ErrInsufficientTier {
		t.Fatalf("expected ErrInsufficientTier, got %v", err)
	}
	if valid.Kind != ClaimKindInference {
		t.Errorf("expected downgrade to inference, got %s", valid.Kind)
	}
}

func TestValidator_ForbiddenCalmCopyViolation(t *testing.T) {
	v := NewValidator()
	snapshot := &EvidenceSnapshot{
		SnapshotVersion: 1,
		CaseID:          "case-123",
		EvidenceItems:   map[string]SnapshotEvidenceItem{},
	}

	claim := Claim{
		Code:        "CLAIM_FAILED",
		Kind:        ClaimKindConfirmed,
		Statement:   "The payment failed due to network timeout.",
		EvidenceIDs: []string{},
	}

	_, err := v.ValidateClaim(claim, snapshot)
	if err == nil {
		t.Fatalf("expected error on forbidden 'payment failed' copy, got nil")
	}
}
