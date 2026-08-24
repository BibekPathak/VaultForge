// VaultForge end-to-end Solana devnet harness.
//
// Builds a REAL Solana SOL-transfer transaction with a real devnet keypair,
// signs it with the real Ed25519 private key, submits it through the
// platform's SolanaClient (base64 encoding, exponential backoff), and
// waits for on-chain confirmation using the platform's poller.
//
// Usage:
//
//	go run ./services/api/cmd/e2e -sender <keypair.json> -to <pubkey> [-amount lamports] [-rpc url]
package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/vaultforge/vaultforge/services/api/core"
)

func main() {
	senderPath := flag.String("sender", "", "path to sender keypair (solana-keygen JSON)")
	recipient := flag.String("to", "", "recipient Solana public key (base58)")
	amountLamports := flag.Uint64("amount", 1_000_000, "amount in lamports to transfer")
	rpc := flag.String("rpc", "https://api.devnet.solana.com", "Solana RPC URL")
	flag.Parse()

	if *senderPath == "" || *recipient == "" {
		log.Fatal("usage: e2e -sender <keypair.json> -to <pubkey> [-amount lamports] [-rpc url]")
	}

	senderKey, err := solana.PrivateKeyFromSolanaKeygenFile(*senderPath)
	if err != nil {
		log.Fatalf("failed to load sender keypair: %v", err)
	}
	senderPub := senderKey.PublicKey()
	recipientPub, err := solana.PublicKeyFromBase58(*recipient)
	if err != nil {
		log.Fatalf("invalid recipient pubkey: %v", err)
	}

	fmt.Printf("sender:    %s\n", senderPub)
	fmt.Printf("recipient: %s\n", recipientPub)
	fmt.Printf("amount:    %d lamports\n", *amountLamports)

	client := core.NewSolanaClient(*rpc)

	blockhashStr, err := client.GetRecentBlockhash()
	if err != nil {
		log.Fatalf("failed to get recent blockhash: %v", err)
	}
	blockhash, err := solana.HashFromBase58(blockhashStr)
	if err != nil {
		log.Fatalf("invalid blockhash from RPC: %v", err)
	}
	fmt.Printf("blockhash: %s\n", blockhashStr)

	transfer := system.NewTransferInstruction(*amountLamports, senderPub, recipientPub)
	tx, err := solana.NewTransaction(
		[]solana.Instruction{transfer.Build()},
		blockhash,
		solana.TransactionPayer(senderPub),
	)
	if err != nil {
		log.Fatalf("failed to build transaction: %v", err)
	}

	if _, err := tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key == senderPub {
			return &senderKey
		}
		return nil
	}); err != nil {
		log.Fatalf("failed to sign transaction: %v", err)
	}

	txBytes, err := tx.MarshalBinary()
	if err != nil {
		log.Fatalf("failed to serialize transaction: %v", err)
	}
	fmt.Printf("tx size:   %d bytes\n", len(txBytes))

	submitStart := time.Now()
	res, err := client.SubmitTransaction(txBytes)
	if err != nil {
		log.Fatalf("submit failed: %v", err)
	}
	if !res.Success {
		log.Fatalf("submit returned error: %s", res.Error)
	}
	fmt.Printf("submitted in %s\n", time.Since(submitStart).Round(time.Millisecond))
	fmt.Printf("signature: %s\n", res.Signature)

	confirmed, err := client.WaitForConfirmation(res.Signature)
	if err != nil {
		log.Fatalf("confirmation failed: %v", err)
	}
	if confirmed {
		fmt.Println("CONFIRMED")
	}
	fmt.Printf("explorer:  https://explorer.solana.com/tx/%s?cluster=devnet\n", res.Signature)
}
