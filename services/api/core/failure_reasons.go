package core

// Failure reason constants used by intent status transitions.
const (
	PolicyDenied          = "policy_denied"
	Expired               = "expired"
	SimulationFailed      = "simulation_failed"
	SigningFailed         = "signing_failed"
	Rejected              = "rejected"
	SubmissionFailed      = "submission_failed"
	ConfirmationTimeout   = "confirmation_timeout"
	BalanceMismatch       = "balance_mismatch"
	PermanentFailure      = "permanent_failure"
)

// FailureReason returns a pointer to the given reason string.
// Used for setting Intent.FailureReason which is *string.
func FailureReason(reason string) *string {
	return &reason
}
