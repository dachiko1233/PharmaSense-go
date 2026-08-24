// Package triplecheck screens signups against an external anti-abuse API
// before an account is created. It calls a provider you configure via env
// (TRIPLECHECK_URL + TRIPLECHECK_API_KEY) and returns an allow/deny decision.
//
// Design notes:
//   - The service is optional. When TRIPLECHECK_URL is empty it is treated as
//     "not configured" and CheckSignup always allows (a no-op), mirroring how
//     the Dodo billing service behaves when its env is unset. This keeps local
//     dev and unconfigured deploys working without the external dependency.
//   - It FAILS OPEN: on a network error, timeout, or non-2xx response the
//     signup is allowed rather than blocked, so a provider outage can never
//     take down registration. The event is logged so failures are visible.
//   - Phone is currently always passed empty — the signup form does not collect
//     a phone number. The field is kept in the request so wiring a phone in
//     later is a form change only, not a contract change.
package triplecheck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Result is the outcome of a signup screen.
type Result struct {
	Allow  bool           `json:"allow"`
	Reason string         `json:"reason"`
	Flags  map[string]any `json:"flags"`
}

// request is the JSON body POSTed to the provider.
type request struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
	IP    string `json:"ip"`
}

// Service is a client for the external screening API.
type Service struct {
	url        string
	apiKey     string
	httpClient *http.Client
}

// New builds a screening client. Pass an empty url to disable screening
// (CheckSignup then always allows). timeout guards each call; if zero a short
// default is used so a slow provider cannot stall the signup request.
func New(url, apiKey string, timeout time.Duration) *Service {
	if timeout <= 0 {
		timeout = 4 * time.Second
	}
	if url != "" {
		slog.Info("triplecheck: signup screening ACTIVE", "url", url, "api_key_set", apiKey != "")
	} else {
		slog.Warn("triplecheck: not configured (TRIPLECHECK_URL empty) — signups will not be screened")
	}
	return &Service{
		url:        url,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// Enabled reports whether a provider URL is configured.
func (s *Service) Enabled() bool { return s != nil && s.url != "" }

// CheckSignup screens a prospective signup. It never returns an error: when
// screening is disabled or the provider is unreachable it fails open, returning
// Allow=true. Callers should deny the signup only when Allow is explicitly
// false. The returned Reason is a user-safe message when Allow is false.
func (s *Service) CheckSignup(ctx context.Context, email, phone, ip string) Result {
	if !s.Enabled() {
		return Result{Allow: true}
	}

	body, err := json.Marshal(request{Email: email, Phone: phone, IP: ip})
	if err != nil {
		// Marshalling a fixed struct should never fail; fail open if it does.
		slog.Error("triplecheck: marshal request failed, allowing signup", "err", err)
		return Result{Allow: true}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(body))
	if err != nil {
		slog.Error("triplecheck: build request failed, allowing signup", "err", err)
		return Result{Allow: true}
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		// Timeout, DNS, connection refused, etc. Fail open.
		slog.Warn("triplecheck: provider unreachable, allowing signup", "err", err)
		return Result{Allow: true}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		slog.Warn("triplecheck: provider returned non-2xx, allowing signup",
			"status", resp.StatusCode, "body", string(snippet))
		return Result{Allow: true}
	}

	var out Result
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		slog.Warn("triplecheck: decode response failed, allowing signup", "err", err)
		return Result{Allow: true}
	}

	if !out.Allow && out.Reason == "" {
		out.Reason = "signup rejected by fraud screening"
	}
	return out
}

// String is a small helper for logging flags compactly.
func (r Result) String() string {
	return fmt.Sprintf("allow=%v reason=%q flags=%v", r.Allow, r.Reason, r.Flags)
}
