// scripts/keygen.go
//go:build tools

// keygen generates a development Ed25519 keypair for provenance signing.
//
// Usage:
//   go run scripts/keygen.go
package main

import (
	"fmt"
	"os"

	"github.com/ZoroNewbie00/kalakriti/pkg/crypto"
)

func main() {
	privHex, pubHex, err := crypto.GenerateKeypair()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating keypair: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Ed25519 Keypair for Provenance Signing ===")
	fmt.Println()
	fmt.Println("Private Key (keep secret, add to .env):")
	fmt.Printf("PROVENANCE_PRIVATE_KEY=%s\n", privHex)
	fmt.Println()
	fmt.Println("Public Key (distribute for verification):")
	fmt.Printf("PROVENANCE_PUBLIC_KEY=%s\n", pubHex)
	fmt.Println()
	fmt.Println("Key ID (set in config):")
	fmt.Println("PROVENANCE_KEY_ID=dev-key-1")
	fmt.Println()
	fmt.Println("⚠️  NEVER commit the private key to version control.")
	fmt.Println("⚠️  Use a proper key management system in production.")
}
