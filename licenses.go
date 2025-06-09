package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

var licenseKey ed25519.PrivateKey

func initLicenses(se *core.ServeEvent) error {
	k := args.LicenseKey
	switch len(k) {
	case 0:
		return se.Next()
	case 32:
		licenseKey = ed25519.NewKeyFromSeed(k)
	case 64:
		licenseKey = ed25519.PrivateKey(k)
	default:
		return fmt.Errorf("--license-key 的值不对, 预期为 ed25519 seed([32]byte) 或 private key([64]byte), 但此key的长度为: %d", len(k))
	}
	return se.Next()
}

func GenLicense(pubkey string) string {
	if len(licenseKey) == 0 {
		return ""
	}
	sig := ed25519.Sign(licenseKey, []byte(pubkey))
	return base64.StdEncoding.EncodeToString(sig)
}
