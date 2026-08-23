package core

import (
	"crypto/sha256"
	"fmt"
	"log"

	"gorm.io/gorm"
)

// MPCSignerImpl implements MPCSigner using 2-of-3 threshold signing.
// In production, this would communicate with 3 MPC nodes via gRPC.
type MPCSignerImpl struct {
	db *gorm.DB
}

func NewMPCSigner(db *gorm.DB) *MPCSignerImpl {
	return &MPCSignerImpl{db: db}
}

// Sign performs threshold signing via MPC protocol.
//
// Flow:
// 1. Load MPC shares for the wallet's tenant
// 2. Compute partial signatures from 2 of 3 shares
// 3. Aggregate into final threshold signature
//
// In production, each partial signing round would be a network call
// to a separate MPC node. Here we simulate the protocol locally.
func (s *MPCSignerImpl) Sign(input SigningInput) (*SigningResult, error) {
	if len(input.IntentHash) == 0 {
		return nil, fmt.Errorf("intent hash is required")
	}
	if input.TxHash == "" {
		return nil, fmt.Errorf("transaction hash is required")
	}

	// Compute the signing message: domain-separated hash of intent + tx
	message := computeSigningMessage(input.IntentHash, []byte(input.TxHash), input.Chain, input.Context)

	// Load MPC shares from database
	var shareRecords []MPCShareRecord
	if err := s.db.Where("share_type = ?", "frost").Find(&shareRecords).Error; err != nil {
		return nil, fmt.Errorf("failed to load MPC shares: %w", err)
	}

	if len(shareRecords) < 2 {
		return nil, fmt.Errorf("insufficient MPC shares: have %d, need 2", len(shareRecords))
	}

	// Simulate partial signing with first 2 shares
	var partialSigs [][]byte
	var participants []uint32

	for i := 0; i < 2 && i < len(shareRecords); i++ {
		partial := computePartialSignature(shareRecords[i].EncryptedShare, message)
		partialSigs = append(partialSigs, partial)
		participants = append(participants, shareRecords[i].ShareID)
	}

	// Aggregate partial signatures into final signature
	signature := aggregatePartialSignatures(partialSigs)

	log.Printf("MPC signing complete: %d participants, %d-byte signature", len(participants), len(signature))

	return &SigningResult{
		Signature:    signature,
		Participants: participants,
	}, nil
}

// computeSigningMessage creates the domain-separated signing message.
func computeSigningMessage(intentHash, txHash []byte, chain, context string) []byte {
	h := sha256.New()
	h.Write([]byte("vaultforge-signing-v1"))
	h.Write([]byte(chain))
	h.Write([]byte(context))
	h.Write(intentHash)
	h.Write(txHash)
	return h.Sum(nil)
}

// computePartialSignature computes a partial signature from a share and message.
// In production, this would use FROST elliptic curve operations.
func computePartialSignature(share []byte, message []byte) []byte {
	h := sha256.New()
	h.Write(share)
	h.Write(message)
	return h.Sum(nil)
}

// aggregatePartialSignatures combines 2 partial signatures into a threshold signature.
// In production, this would use Lagrange interpolation over the participant identifiers.
func aggregatePartialSignatures(partials [][]byte) []byte {
	h := sha256.New()
	h.Write([]byte("vaultforge-aggregate-v1"))
	for _, p := range partials {
		h.Write(p)
	}
	result := h.Sum(nil)
	// Pad to 64 bytes (Ed25519 signature format)
	sig := make([]byte, 64)
	copy(sig, result)
	return sig
}
