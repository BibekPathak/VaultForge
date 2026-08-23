package core

import (
	"testing"
)

func TestFailureReason_ReturnsPointer(t *testing.T) {
	reason := FailureReason(PolicyDenied)
	if reason == nil {
		t.Fatal("FailureReason should return non-nil pointer")
	}
	if *reason != PolicyDenied {
		t.Errorf("expected %q, got %q", PolicyDenied, *reason)
	}
}

func TestFailureReason_AllConstants(t *testing.T) {
	constants := []string{
		PolicyDenied,
		Expired,
		SimulationFailed,
		SigningFailed,
		Rejected,
		SubmissionFailed,
		ConfirmationTimeout,
		BalanceMismatch,
		PermanentFailure,
	}
	for _, c := range constants {
		r := FailureReason(c)
		if r == nil || *r != c {
			t.Errorf("FailureReason(%q) failed", c)
		}
	}
}

func TestFailureReasonStringValues(t *testing.T) {
	if PolicyDenied != "policy_denied" {
		t.Errorf("PolicyDenied = %q", PolicyDenied)
	}
	if Expired != "expired" {
		t.Errorf("Expired = %q", Expired)
	}
	if SimulationFailed != "simulation_failed" {
		t.Errorf("SimulationFailed = %q", SimulationFailed)
	}
	if SigningFailed != "signing_failed" {
		t.Errorf("SigningFailed = %q", SigningFailed)
	}
	if Rejected != "rejected" {
		t.Errorf("Rejected = %q", Rejected)
	}
	if SubmissionFailed != "submission_failed" {
		t.Errorf("SubmissionFailed = %q", SubmissionFailed)
	}
	if ConfirmationTimeout != "confirmation_timeout" {
		t.Errorf("ConfirmationTimeout = %q", ConfirmationTimeout)
	}
	if BalanceMismatch != "balance_mismatch" {
		t.Errorf("BalanceMismatch = %q", BalanceMismatch)
	}
	if PermanentFailure != "permanent_failure" {
		t.Errorf("PermanentFailure = %q", PermanentFailure)
	}
}
