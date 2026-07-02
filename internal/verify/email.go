package verify

import (
	"fmt"
	"log"
	"strings"

	emailverifier "github.com/AfterShip/email-verifier"
)

// Method determines how strictly we validate an email address.
type Method string

const (
	MethodDefault Method = "default" // syntax + MX + disposable + role (no SMTP)
	MethodSMTP    Method = "smtp"    // full SMTP mailbox check on port 25
	MethodOff     Method = "off"     // skip verification entirely
)

// Result holds the outcome of an email verification.
type Result struct {
	Email     string `json:"email"`
	Valid     bool   `json:"valid"`
	Reason    string `json:"reason,omitempty"`
	Reachable string `json:"reachable,omitempty"` // yes / no / unknown
}

// Verifier validates email addresses using the configured method.
type Verifier struct {
	method   Method
	verifier *emailverifier.Verifier
}

// New returns a Verifier for the given method.
func New(method Method) *Verifier {
	v := &Verifier{method: method}

	switch method {
	case MethodDefault:
		// Syntax + MX + disposable domain + role account checks.
		// No SMTP — works on any network.
		v.verifier = emailverifier.
			NewVerifier().
			EnableDomainSuggest().
			EnableAutoUpdateDisposable()
	case MethodSMTP:
		// Full SMTP mailbox verification on port 25.
		// Requires port 25 to be open. Will time out if blocked.
		v.verifier = emailverifier.
			NewVerifier().
			EnableSMTPCheck().
			EnableDomainSuggest().
			EnableAutoUpdateDisposable()
	case MethodOff:
		// No verifier needed — always passes.
	}

	return v
}

// Verify checks an email address and returns a Result.
func (v *Verifier) Verify(email string) *Result {
	r := &Result{Email: email, Valid: true}

	if v.method == MethodOff {
		return r
	}

	if v.verifier == nil {
		log.Printf("verify: no verifier configured for method %q", v.method)
		return r
	}

	result, err := v.verifier.Verify(email)
	if err != nil {
		// DNS failures: the domain literally does not exist.
		if strings.Contains(err.Error(), "no such host") ||
			strings.Contains(err.Error(), "NXDOMAIN") {
			domain := extractDomain(email)
			r.Valid = false
			r.Reason = fmt.Sprintf("domain %q does not exist", domain)
			return r
		}

		// SMTP network failures: port 25 blocked, timed out, connection refused.
		// These are NOT the email's fault — our network can't reach the mail server.
		// Log it but let the email through.
		if v.method == MethodSMTP && isNetworkError(err) {
			log.Printf("verify: %s: SMTP unreachable (port 25 blocked?), falling back: %v", email, err)
			r.Valid = true
			r.Reachable = "unknown"
			return r
		}

		// Other errors — let it through rather than blocking real users.
		log.Printf("verify: %s: %v", email, err)
		r.Valid = true
		r.Reachable = "unknown"
		return r
	}

	// Syntax check — most basic, always reject if invalid.
	if !result.Syntax.Valid {
		r.Valid = false
		r.Reason = "invalid email syntax"
		return r
	}

	// Disposable email addresses — reject outright.
	if result.Disposable {
		r.Valid = false
		r.Reason = "disposable email addresses are not allowed"
		return r
	}

	// MX records — reject if domain has no mail server.
	if !result.HasMxRecords {
		domain := extractDomain(email)
		r.Valid = false
		r.Reason = fmt.Sprintf("domain %q has no MX records — cannot receive email", domain)
		return r
	}

	// SMTP reachability (only when method is "smtp" and the call succeeded).
	if v.method == MethodSMTP {
		r.Reachable = result.Reachable
		if result.Reachable == "no" {
			r.Valid = false
			r.Reason = "email address is not reachable"
			return r
		}
	}

	return r
}

func extractDomain(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return email
}

func isNetworkError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connect:") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "network is unreachable") ||
		strings.Contains(msg, "no route to host")
}
