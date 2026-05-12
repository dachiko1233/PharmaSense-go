package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/resend/resend-go/v2"
)

type EmailService struct {
	client    *resend.Client
	fromEmail string
	isMock    bool
}

func NewEmailService(apiKey, fromEmail string) *EmailService {
	isMock := apiKey == "" || strings.HasPrefix(apiKey, "mock_")
	if isMock {
		return &EmailService{fromEmail: fromEmail, isMock: true}
	}
	return &EmailService{
		client:    resend.NewClient(apiKey),
		fromEmail: fromEmail,
		isMock:    false,
	}
}

func (s *EmailService) Send(ctx context.Context, to, subject, html string) error {
	if s.isMock {
		slog.Info("[MOCK EMAIL]",
			"to", to,
			"subject", subject,
			"body_preview", truncate(html, 100),
		)
		return nil
	}

	params := &resend.SendEmailRequest{
		From:    s.fromEmail,
		To:      []string{to},
		Subject: subject,
		Html:    html,
	}
	_, err := s.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}

func (s *EmailService) SendWelcome(ctx context.Context, to, fullName, appURL, verificationToken string) error {
	verifyLink := fmt.Sprintf("%s/verify-email?token=%s", appURL, verificationToken)
	html := fmt.Sprintf(`
<div style="font-family:sans-serif;max-width:600px;margin:0 auto">
  <h1 style="color:#059669">Welcome to PharmaSense, %s!</h1>
  <p>Thank you for signing up. PharmaSense helps your pharmacy monitor expiry dates and reduce waste.</p>
  <p>Please verify your email to unlock all features:</p>
  <a href="%s" style="display:inline-block;background:#059669;color:white;padding:12px 24px;border-radius:6px;text-decoration:none;font-weight:600">Verify Email</a>
  <p style="color:#6b7280;font-size:14px;margin-top:32px">This link expires in 24 hours. If you didn't create an account, you can safely ignore this email.</p>
</div>`, fullName, verifyLink)
	return s.Send(ctx, to, "Welcome to PharmaSense — Verify Your Email", html)
}

func (s *EmailService) SendPasswordReset(ctx context.Context, to, fullName, appURL, resetToken string) error {
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", appURL, resetToken)
	html := fmt.Sprintf(`
<div style="font-family:sans-serif;max-width:600px;margin:0 auto">
  <h1 style="color:#059669">Password Reset — PharmaSense</h1>
  <p>Hello %s,</p>
  <p>We received a request to reset your password. Click the button below:</p>
  <a href="%s" style="display:inline-block;background:#059669;color:white;padding:12px 24px;border-radius:6px;text-decoration:none;font-weight:600">Reset Password</a>
  <p style="color:#6b7280;font-size:14px;margin-top:32px">This link expires in 1 hour. If you didn't request a reset, please ignore this email.</p>
</div>`, fullName, resetLink)
	return s.Send(ctx, to, "Reset Your PharmaSense Password", html)
}

func (s *EmailService) SendDailyDigest(ctx context.Context, to, fullName, pharmacyName, appURL string, criticalCount int, estimatedLoss float64) error {
	html := fmt.Sprintf(`
<div style="font-family:sans-serif;max-width:600px;margin:0 auto">
  <h1 style="color:#059669">Daily Expiry Digest — %s</h1>
  <p>Hello %s,</p>
  <div style="background:#fef2f2;border:1px solid #fecaca;border-radius:8px;padding:16px;margin:16px 0">
    <strong style="color:#dc2626">⚠️ %d CRITICAL items</strong> are expiring within 30 days.
    <br>Estimated potential loss: <strong>€%.2f</strong>
  </div>
  <p>Log in to PharmaSense to take action on these items before they expire.</p>
  <a href="%s/dashboard" style="display:inline-block;background:#059669;color:white;padding:12px 24px;border-radius:6px;text-decoration:none;font-weight:600">View Dashboard</a>
  <p style="color:#6b7280;font-size:14px;margin-top:32px">You're receiving this because you have daily digest notifications enabled.</p>
</div>`, pharmacyName, fullName, criticalCount, estimatedLoss, appURL)
	return s.Send(ctx, to, fmt.Sprintf("⚠️ %d Critical Items — PharmaSense Daily Digest", criticalCount), html)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
