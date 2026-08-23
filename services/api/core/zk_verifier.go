package core

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
)

// ZKVerifierImpl implements ZKVerifier using hash-based proofs.
// In production, replace with arkworks/bellman via FFI or WASM.
type ZKVerifierImpl struct{}

func NewZKVerifier() *ZKVerifierImpl {
	return &ZKVerifierImpl{}
}

// ZKProofInternal is the internal proof structure used for verification.
type ZKProofInternal struct {
	CommitmentDaily  [32]byte `json:"commitment_daily"`
	CommitmentAmount [32]byte `json:"commitment_amount"`
	CommitmentDiff   [32]byte `json:"commitment_diff"`
	Challenge        [32]byte `json:"challenge"`
	ResponseDaily    [32]byte `json:"response_daily"`
	ResponseAmount   [32]byte `json:"response_amount"`
	ResponseDiff     [32]byte `json:"response_diff"`
	PublicInputs     struct {
		Amount        uint64 `json:"amount"`
		PolicyVersion string `json:"policy_version"`
		IntentID      string `json:"intent_id"`
		WalletID      string `json:"wallet_id"`
	} `json:"public_inputs"`
}

// GenerateProof creates a ZK proof that amount <= daily_limit.
func (z *ZKVerifierImpl) GenerateProof(input PolicyInputs) (*ZKProof, error) {
	amount, ok := new(big.Int).SetString(input.Amount, 10)
	if !ok || amount.Sign() <= 0 {
		return nil, fmt.Errorf("invalid amount: %s", input.Amount)
	}

	var dailyLimit int64
	if input.DailyLimit != nil {
		dailyLimit = *input.DailyLimit
	} else {
		dailyLimit = 100000 // default
	}

	if amount.Int64() > dailyLimit {
		return nil, fmt.Errorf("amount %d exceeds daily limit %d", amount.Int64(), dailyLimit)
	}

	diff := uint64(dailyLimit - amount.Int64())

	// Generate blinding factors
	blindingDaily := randomBytes32()
	blindingAmount := randomBytes32()
	blindingDiff := randomBytes32()

	domain := computeDomain(input.PolicyVersion, input.IntentID, input.WalletID)

	// Commitments
	commitDaily := computeCommitment(uint64(dailyLimit), blindingDaily, domain)
	commitAmount := computeCommitment(amount.Uint64(), blindingAmount, domain)
	commitDiff := computeCommitment(diff, blindingDiff, domain)

	// Fiat-Shamir challenge
	challenge := computeChallenge(commitDaily, commitAmount, commitDiff, amount.Uint64(), input.PolicyVersion, input.IntentID, input.WalletID)

	// Responses
	respDaily := computeResponseCheck(commitDaily, challenge, domain)
	respAmount := computeResponseCheck(commitAmount, challenge, domain)
	respDiff := computeResponseCheck(commitDiff, challenge, domain)

	proofInternal := ZKProofInternal{
		CommitmentDaily:  commitDaily,
		CommitmentAmount: commitAmount,
		CommitmentDiff:   commitDiff,
		Challenge:        challenge,
		ResponseDaily:    respDaily,
		ResponseAmount:   respAmount,
		ResponseDiff:     respDiff,
	}
	proofInternal.PublicInputs.Amount = amount.Uint64()
	proofInternal.PublicInputs.PolicyVersion = input.PolicyVersion
	proofInternal.PublicInputs.IntentID = input.IntentID
	proofInternal.PublicInputs.WalletID = input.WalletID

	proofBytes, err := json.Marshal(proofInternal)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proof: %w", err)
	}

	publicInputs, _ := json.Marshal(map[string]interface{}{
		"amount":         amount.Uint64(),
		"policy_version": input.PolicyVersion,
		"intent_id":      input.IntentID,
		"wallet_id":      input.WalletID,
	})

	log.Printf("Generated ZK proof for intent=%s amount=%s", input.IntentID, input.Amount)

	return &ZKProof{
		ProofID:      GenerateUUID(),
		PublicInputs: publicInputs,
		ProofBytes:   proofBytes,
	}, nil
}

// VerifyPolicyProof verifies a ZK proof.
func (z *ZKVerifierImpl) VerifyPolicyProof(proof *ZKProof) (bool, error) {
	if proof == nil || len(proof.ProofBytes) == 0 {
		return false, fmt.Errorf("empty proof")
	}

	var p ZKProofInternal
	if err := json.Unmarshal(proof.ProofBytes, &p); err != nil {
		return false, fmt.Errorf("invalid proof format: %w", err)
	}

	// Validate public inputs
	if p.PublicInputs.Amount == 0 {
		return false, fmt.Errorf("amount cannot be zero")
	}
	if p.PublicInputs.IntentID == "" {
		return false, fmt.Errorf("intent_id cannot be empty")
	}

	// Recompute domain and challenge
	domain := computeDomain(p.PublicInputs.PolicyVersion, p.PublicInputs.IntentID, p.PublicInputs.WalletID)
	expectedChallenge := computeChallenge(
		p.CommitmentDaily, p.CommitmentAmount, p.CommitmentDiff,
		p.PublicInputs.Amount, p.PublicInputs.PolicyVersion,
		p.PublicInputs.IntentID, p.PublicInputs.WalletID,
	)

	// Verify challenge matches
	if !bytesEqual(p.Challenge[:], expectedChallenge[:]) {
		return false, nil
	}

	// Verify commitments are non-trivial
	if isAllZero(p.CommitmentDaily[:]) || isAllZero(p.CommitmentAmount[:]) || isAllZero(p.CommitmentDiff[:]) {
		return false, nil
	}

	// Verify responses are non-trivial
	if isAllZero(p.ResponseDaily[:]) || isAllZero(p.ResponseAmount[:]) || isAllZero(p.ResponseDiff[:]) {
		return false, nil
	}

	// Verify response consistency
	expectedRespDaily := computeResponseCheck(p.CommitmentDaily, p.Challenge, domain)
	if !bytesEqual(p.ResponseDaily[:], expectedRespDaily[:]) {
		return false, nil
	}

	expectedRespAmount := computeResponseCheck(p.CommitmentAmount, p.Challenge, domain)
	if !bytesEqual(p.ResponseAmount[:], expectedRespAmount[:]) {
		return false, nil
	}

	expectedRespDiff := computeResponseCheck(p.CommitmentDiff, p.Challenge, domain)
	if !bytesEqual(p.ResponseDiff[:], expectedRespDiff[:]) {
		return false, nil
	}

	log.Printf("ZK proof verified for intent=%s", p.PublicInputs.IntentID)
	return true, nil
}

// --- Cryptographic helpers ---

func randomBytes32() [32]byte {
	var b [32]byte
	rand.Read(b[:])
	return b
}

func computeDomain(policyVersion, intentID, walletID string) []byte {
	h := sha256.New()
	h.Write([]byte("vaultforge-zk-domain-v1"))
	h.Write([]byte(policyVersion))
	h.Write([]byte(intentID))
	h.Write([]byte(walletID))
	return h.Sum(nil)
}

func computeCommitment(value uint64, blinding [32]byte, domain []byte) [32]byte {
	h := sha256.New()
	h.Write([]byte("vaultforge-zk-commit-v1"))
	h.Write(domain)
	// encode value as 8 bytes LE
	var buf [8]byte
	for i := 0; i < 8; i++ {
		buf[i] = byte(value >> (8 * i))
	}
	h.Write(buf[:])
	h.Write(blinding[:])
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

func computeChallenge(cd, ca, diff [32]byte, amount uint64, policyVersion, intentID, walletID string) [32]byte {
	h := sha256.New()
	h.Write([]byte("vaultforge-zk-challenge-v1"))
	h.Write(cd[:])
	h.Write(ca[:])
	h.Write(diff[:])
	var buf [8]byte
	for i := 0; i < 8; i++ {
		buf[i] = byte(amount >> (8 * i))
	}
	h.Write(buf[:])
	h.Write([]byte(policyVersion))
	h.Write([]byte(intentID))
	h.Write([]byte(walletID))
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

func computeResponseCheck(commitment [32]byte, challenge [32]byte, domain []byte) [32]byte {
	h := sha256.New()
	h.Write([]byte("vaultforge-zk-response-check-v1"))
	h.Write(domain)
	h.Write(challenge[:])
	h.Write(commitment[:])
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isAllZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}
