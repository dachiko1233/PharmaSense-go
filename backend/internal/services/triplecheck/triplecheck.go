// Package triplecheck screens signups against the TripleCheck anti-abuse API
// (https://triplecheck.it.com) before an account is created. Server-side only —
// the API key must never reach the browser.
//
// Design notes:
//   - The service is optional. When no API key is configured it is treated as
//     "not configured" and CheckSignup always allows (a no-op that makes NO
//     network calls), mirroring how the Dodo billing service behaves when its
//     env is unset. This keeps local dev and unconfigured deploys working —
//     and, since each check costs a credit, avoids burning credits in dev.
//   - It FAILS OPEN: a network error, timeout, or non-200 response makes the
//     individual check return nil, which is skipped. Only an explicit "invalid"
//     status denies the signup. "risky"/"unknown" never turn a real customer
//     away — they are surfaced as flags for review.
//   - Phone is currently always passed empty — the signup form does not collect
//     a phone number, so the phone check is simply not run. Wiring a phone in
//     later is a form change only, not a contract change.
package triplecheck

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// base is the TripleCheck verify endpoint root. Individual checks are POSTed to
// base + "/" + kind (e.g. .../verify/email). A Service may override this (for
// tests/staging); when its override is empty this default is used.
const base = "https://api.triplecheck.it.com/v1/verify"

// Result is a single TripleCheck verification response.
type Result struct {
	Type             string         `json:"type"`
	Input            string         `json:"input"`
	Status           string         `json:"status"` // "valid" | "invalid" | "risky" | "unknown"
	Reason           string         `json:"reason"`
	Details          map[string]any `json:"details"`
	LookupID         string         `json:"lookupId"`
	CreditsRemaining int            `json:"creditsRemaining"`
}

// Flag is a non-fatal signal worth reviewing (a "risky" check).
type Flag struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

// SignupCheck is the aggregate decision across the checks that were run.
type SignupCheck struct {
	Allow  bool      `json:"allow"`
	Reason string    `json:"reason"`
	Flags  []Flag    `json:"flags"`
	Checks []*Result `json:"checks"`
}

// Service is a client for the TripleCheck verify API.
type Service struct {
	base       string
	apiKey     string
	httpClient *http.Client
}

// New builds a screening client. Pass an empty apiKey to disable screening
// (CheckSignup then always allows and makes no network calls). baseOverride is
// optional — leave it empty to use the production base. timeout guards each
// call; if zero a default is used so a slow provider cannot stall signup.
func New(baseOverride, apiKey string, timeout time.Duration) *Service {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	b := baseOverride
	if b == "" {
		b = base
	}
	if apiKey != "" {
		slog.Info("triplecheck: signup screening ACTIVE", "base", b)
	} else {
		slog.Warn("triplecheck: not configured (TRIPLECHECK_KEY empty) — signups will not be screened")
	}
	return &Service{
		base:       b,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// Enabled reports whether an API key is configured.
func (s *Service) Enabled() bool { return s != nil && s.apiKey != "" }

// verify POSTs a single check. It returns nil on any error or non-200 response
// so the caller can fail open. TripleCheck does not bill 4xx/5xx, so a nil here
// costs nothing.
func (s *Service) verify(ctx context.Context, kind string, body map[string]string) *Result {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/"+kind, bytes.NewReader(payload))
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := s.httpClient.Do(req)
	if err != nil {
		slog.Warn("triplecheck: check failed, skipping", "kind", kind, "err", err)
		return nil
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		slog.Warn("triplecheck: check returned non-200, skipping", "kind", kind, "status", res.StatusCode)
		return nil
	}

	var out Result
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		slog.Warn("triplecheck: decode failed, skipping", "kind", kind, "err", err)
		return nil
	}
	return &out
}

// CheckSignup runs the applicable checks in parallel (one credit per check
// performed) and aggregates them. When screening is disabled it allows without
// any network calls.
//
// Only an "invalid" status denies the signup. "risky" is recorded as a flag and
// "unknown"/failed checks are ignored — neither is a reason to reject a real
// customer.
func (s *Service) CheckSignup(ctx context.Context, email, phone, ip string) SignupCheck {
	out := SignupCheck{Allow: true, Flags: []Flag{}, Checks: []*Result{}}
	if !s.Enabled() {
		return out
	}

	var wg sync.WaitGroup
	results := make([]*Result, 3)

	run := func(i int, fn func() *Result) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = fn()
		}()
	}

	if email != "" {
		run(0, func() *Result { return s.verify(ctx, "email", map[string]string{"email": email}) })
	}
	if phone != "" {
		run(1, func() *Result { return s.verify(ctx, "phone", map[string]string{"phone": phone}) })
	}
	if ip != "" {
		run(2, func() *Result { return s.verify(ctx, "ip", map[string]string{"ip": ip}) })
	}
	wg.Wait()

	for _, r := range results {
		if r == nil {
			continue
		}
		out.Checks = append(out.Checks, r)

		switch r.Status {
		case "invalid":
			if out.Allow {
				out.Allow = false
				out.Reason = r.Reason
			}
		case "risky":
			out.Flags = append(out.Flags, Flag{Type: r.Type, Reason: r.Reason})
		}
	}

	if !out.Allow && out.Reason == "" {
		out.Reason = "signup rejected by fraud screening"
	}
	return out
}
