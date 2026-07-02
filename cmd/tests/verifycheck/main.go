package main

import (
	"fmt"
	"log"

	"Communicate/internal/config"
	"Communicate/internal/verify"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	fmt.Printf("Email verifier method: %s\n", cfg.EmailVerifierMethod)

	if cfg.EmailVerifierMethod == "off" {
		fmt.Println("SKIP: email verification is disabled, nothing to test")
		return
	}

	verifier := verify.New(verify.Method(cfg.EmailVerifierMethod))
	fmt.Println()

	// Test cases covering all rejection categories.
	tests := []struct {
		email string
		label string
	}{
		// Should pass
		{"user@gmail.com", "valid gmail"},
		{"user@outlook.com", "valid outlook"},
		{"user@yahoo.com", "valid yahoo"},
		{"user@protonmail.com", "valid protonmail"},
		{"dhruvadeepmalakar12345@gmail.com", "real gmail address"},
		{"spam@dhruvadeep.app", "real domain with mail (dhruvadeep.app)"},

		// Should pass with "default" (MX exists), would fail with "smtp" (mailbox doesn't exist)
		{"idk123@dhruvadeep.app", "fake mailbox, real domain (dhruvadeep.app)"},

		// Should fail: bad syntax
		{"not-an-email", "bad syntax"},
		{"missing@tld", "missing TLD"},
		{"@no-username.com", "no username"},

		// Should fail: disposable (if library data is available)
		{"user@mailinator.com", "disposable (mailinator)"},
		{"user@guerrillamail.com", "disposable (guerrillamail)"},
		{"user@10minutemail.com", "disposable (10minutemail)"},

		// Should pass: role-looking usernames (we don't reject these)
		{"admin@gmail.com", "admin account"},
		{"noreply@gmail.com", "noreply account"},
		{"support@gmail.com", "support account"},

		// Should fail: no MX records (bogus domain)
		{"user@this-domain-definitely-does-not-exist-12345.com", "no MX (bogus domain)"},

		// Edge cases
		{"user@gmail", "missing TLD (gmail)"},
		{"", "empty string"},
	}

	passed := 0
	failed := 0

	for _, tc := range tests {
		result := verifier.Verify(tc.email)

		status := "PASS"
		if !result.Valid {
			status = "FAIL"
			failed++
		} else {
			passed++
		}

		reason := ""
		if result.Reason != "" {
			reason = " — " + result.Reason
		}

		fmt.Printf("  [%s] %-20s | %-50s%s\n", status, tc.label, tc.email, reason)
	}

	fmt.Printf("\n─── Results ───\n")
	fmt.Printf("  passed: %d\n", passed)
	fmt.Printf("  failed: %d\n", failed)
	fmt.Printf("  total:  %d\n", passed+failed)
}
