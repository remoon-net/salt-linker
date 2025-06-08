package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/shynome/err0/try"
)

func TestMain(m *testing.M) {
	_, licenseKey = try.To2(ed25519.GenerateKey(rand.Reader))
	m.Run()
}

func TestGenLicense(t *testing.T) {
	vm := goja.New()
	registry := require.NewRegistry()
	registry.Enable(vm)
	console.Enable(vm)

	try.To(vm.Set("GenLicense", GenLicense))

	msg := "xxxx"

	v, err := vm.RunString(fmt.Sprintf(`GenLicense("%s")`, msg))
	if err != nil {
		t.Error(err)
		return
	}
	s1 := v.String()
	s2 := GenLicense(msg)
	if s1 != s2 {
		t.Error(s1, s2)
		return
	}

	sig := try.To1(base64.RawStdEncoding.DecodeString(s1))
	pubkey := licenseKey.Public().(ed25519.PublicKey)
	valid := ed25519.Verify(pubkey, []byte(msg), sig)
	if !valid {
		t.Error("verify signature failed")
		return
	}

	pubkeyStr := base64.RawStdEncoding.EncodeToString(pubkey)
	t.Log(pubkeyStr)
}
