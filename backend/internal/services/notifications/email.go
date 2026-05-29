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

// DigestItem is one at-risk product row in the daily digest email.
type DigestItem struct {
	ProductName string
	DaysLeft    int
	Quantity    int
	Loss        float64
	RiskLevel   string
}

// DigestData holds all data needed to render a daily digest email.
type DigestData struct {
	FullName         string
	PharmacyName     string
	Language         string // "en" or "el"
	AppURL           string
	ExpiredCount     int
	WeekCount        int     // CRITICAL items expiring within 7 days
	CriticalCount    int     // all CRITICAL items (0–30 days)
	HighCount        int     // HIGH items (31–90 days)
	TotalAtRisk      float64 // sum of estimated_loss for EXPIRED+CRITICAL+HIGH
	PotentialSavings float64 // recoverable value by acting today
	TopItems         []DigestItem
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
			"body_preview", truncate(html, 120),
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
	verifyLink := fmt.Sprintf("%s/en/verify-email?token=%s", appURL, verificationToken)
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
	resetLink := fmt.Sprintf("%s/en/reset-password?token=%s", appURL, resetToken)
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

func (s *EmailService) SendDailyDigest(ctx context.Context, to string, data DigestData) error {
	subject, html := buildDigestEmail(data)
	return s.Send(ctx, to, subject, html)
}

// buildDigestEmail generates the subject line and full HTML body for the daily digest.
func buildDigestEmail(d DigestData) (subject, html string) {
	isEl := d.Language == "el"
	total := d.ExpiredCount + d.CriticalCount + d.HighCount

	// ── subject ────────────────────────────────────────────────────
	if total == 0 {
		if isEl {
			subject = fmt.Sprintf("PharmaSense: ✅ Όλα Καλά — %s", d.PharmacyName)
		} else {
			subject = fmt.Sprintf("PharmaSense: ✅ All Clear — %s", d.PharmacyName)
		}
	} else {
		if isEl {
			subject = fmt.Sprintf("PharmaSense Ημερήσια Ειδοποίηση: %d είδη χρειάζονται προσοχή — %s", total, d.PharmacyName)
		} else {
			subject = fmt.Sprintf("PharmaSense Daily Alert: %d items need attention — %s", total, d.PharmacyName)
		}
	}

	// ── localised strings ──────────────────────────────────────────
	greeting := "Good morning"
	hereIs := "Here's your daily inventory health check for"
	summaryTitle := "📊 Today's Summary"
	expiredLabel := "EXPIRED items — REMOVE IMMEDIATELY"
	weekLabel := "items expiring this week"
	criticalLabel := "CRITICAL items (within 30 days)"
	highLabel := "HIGH RISK items (within 90 days)"
	atRiskLabel := "Total at risk"
	savingsLabel := "Potential savings if acted today"
	priorityTitle := "🎯 Top 5 Priority Items"
	viewBtn := "View All Alerts"
	managePrefs := "Manage email preferences"
	haveAGreatDay := "Have a great day!"
	teamSig := "PharmaSense Team"
	daysLeft := "days left"
	units := "units"
	allClearTitle := "🎉 Great news!"
	allClearLine1 := "No critical inventory issues today."
	allClearLine2 := "Your pharmacy is well-managed."
	expiredWord := "EXPIRED"

	if isEl {
		greeting = "Καλημέρα"
		hereIs = "Εδώ είναι η καθημερινή επισκόπηση αποθέματος για"
		summaryTitle = "📊 Σύνοψη Σήμερα"
		expiredLabel = "ΛΗΓΜΕΝΑ είδη — ΑΦΑΙΡΕΣΤΕ ΑΜΕΣΩΣ"
		weekLabel = "είδη που λήγουν αυτή την εβδομάδα"
		criticalLabel = "ΚΡΙΣΙΜΑ είδη (εντός 30 ημερών)"
		highLabel = "ΥΨΗΛΟΥ ΚΙΝΔΥΝΟΥ είδη (εντός 90 ημερών)"
		atRiskLabel = "Συνολικά σε κίνδυνο"
		savingsLabel = "Πιθανή εξοικονόμηση αν ενεργήσετε σήμερα"
		priorityTitle = "🎯 5 Κορυφαία Προϊόντα Προτεραιότητας"
		viewBtn = "Δείτε Όλες τις Ειδοποιήσεις"
		managePrefs = "Διαχείριση προτιμήσεων email"
		haveAGreatDay = "Καλή συνέχεια!"
		teamSig = "PharmaSense Team"
		daysLeft = "ημέρες"
		units = "μον."
		allClearTitle = "🎉 Υπέροχα νέα!"
		allClearLine1 = "Δεν υπάρχουν κρίσιμα προβλήματα αποθέματος σήμερα."
		allClearLine2 = "Το φαρμακείο σας είναι καλά διαχειριζόμενο."
		expiredWord = "ΛΗΓΜΕΝΟ"
	}

	alertsURL := fmt.Sprintf("%s/en/alerts", d.AppURL)
	settingsURL := fmt.Sprintf("%s/en/settings", d.AppURL)
	if isEl {
		alertsURL = fmt.Sprintf("%s/el/alerts", d.AppURL)
		settingsURL = fmt.Sprintf("%s/el/settings", d.AppURL)
	}

	// ── build body section ─────────────────────────────────────────
	var bodyHTML strings.Builder

	if total == 0 {
		// All-clear message
		fmt.Fprintf(&bodyHTML, `
      <p style="margin:0 0 24px;font-size:16px;color:#374151;">%s, <strong>%s</strong>!</p>
      <div style="text-align:center;padding:40px 24px;background:#f0fdf4;border:1px solid #bbf7d0;border-radius:10px;">
        <p style="font-size:48px;margin:0 0 8px;line-height:1;">🎉</p>
        <p style="font-size:20px;font-weight:700;color:#059669;margin:0 0 10px;">%s</p>
        <p style="font-size:15px;color:#374151;margin:0 0 4px;">%s</p>
        <p style="font-size:15px;color:#374151;margin:0;">%s</p>
      </div>`,
			greeting, d.FullName,
			allClearTitle, allClearLine1, allClearLine2,
		)
	} else {
		// Greeting + pharmacy name
		fmt.Fprintf(&bodyHTML, `
      <p style="margin:0 0 6px;font-size:16px;color:#374151;">%s, <strong>%s</strong>!</p>
      <p style="margin:0 0 24px;font-size:14px;color:#64748b;">%s <strong>%s</strong>.</p>`,
			greeting, d.FullName, hereIs, d.PharmacyName,
		)

		// Summary card
		fmt.Fprintf(&bodyHTML, `
      <div style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:10px;padding:20px;margin-bottom:20px;">
        <p style="margin:0 0 14px;font-size:12px;font-weight:700;color:#64748b;text-transform:uppercase;letter-spacing:0.8px;">%s</p>
        <table role="presentation" width="100%%" cellpadding="0" cellspacing="0">`, summaryTitle)

		if d.ExpiredCount > 0 {
			fmt.Fprintf(&bodyHTML, `
          <tr>
            <td style="padding:8px 0;border-bottom:1px solid #e2e8f0;font-size:14px;color:#374151;">
              🔴 <strong style="color:#dc2626;">%d %s</strong>
            </td>
          </tr>`, d.ExpiredCount, expiredLabel)
		}
		if d.WeekCount > 0 {
			fmt.Fprintf(&bodyHTML, `
          <tr>
            <td style="padding:8px 0;border-bottom:1px solid #e2e8f0;font-size:14px;color:#374151;">
              ⚠️ <strong style="color:#d97706;">%d %s</strong>
            </td>
          </tr>`, d.WeekCount, weekLabel)
		}
		if d.CriticalCount > 0 {
			fmt.Fprintf(&bodyHTML, `
          <tr>
            <td style="padding:8px 0;border-bottom:1px solid #e2e8f0;font-size:14px;color:#374151;">
              🟠 <strong style="color:#ea580c;">%d %s</strong>
            </td>
          </tr>`, d.CriticalCount, criticalLabel)
		}
		if d.HighCount > 0 {
			fmt.Fprintf(&bodyHTML, `
          <tr>
            <td style="padding:8px 0;font-size:14px;color:#374151;">
              🟡 <strong style="color:#ca8a04;">%d %s</strong>
            </td>
          </tr>`, d.HighCount, highLabel)
		}

		bodyHTML.WriteString(`
        </table>
      </div>`)

		// Financial impact
		if d.TotalAtRisk > 0 {
			fmt.Fprintf(&bodyHTML, `
      <div style="background:#fff7ed;border:1px solid #fed7aa;border-radius:10px;padding:16px 20px;margin-bottom:20px;">
        <p style="margin:0 0 6px;font-size:14px;color:#374151;">💰 <strong>%s:</strong> €%.2f</p>
        <p style="margin:0;font-size:14px;color:#374151;">💡 <strong>%s:</strong> €%.2f</p>
      </div>`, atRiskLabel, d.TotalAtRisk, savingsLabel, d.PotentialSavings)
		}

		// Top items
		if len(d.TopItems) > 0 {
			fmt.Fprintf(&bodyHTML, `
      <p style="margin:0 0 10px;font-size:12px;font-weight:700;color:#64748b;text-transform:uppercase;letter-spacing:0.8px;">%s</p>
      <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="border:1px solid #e2e8f0;border-radius:10px;overflow:hidden;">`,
				priorityTitle)

			for i, item := range d.TopItems {
				borderBottom := "border-bottom:1px solid #e2e8f0;"
				if i == len(d.TopItems)-1 {
					borderBottom = ""
				}
				bg := "#ffffff"
				if i%2 == 1 {
					bg = "#f8fafc"
				}

				var riskBadge string
				var dayText string
				if item.RiskLevel == "EXPIRED" {
					riskBadge = fmt.Sprintf(`<span style="display:inline-block;background:#fef2f2;color:#dc2626;font-size:11px;font-weight:700;padding:2px 7px;border-radius:4px;">%s</span>`, expiredWord)
					dayText = expiredWord
				} else {
					switch item.RiskLevel {
					case "CRITICAL":
						riskBadge = fmt.Sprintf(`<span style="display:inline-block;background:#fff7ed;color:#ea580c;font-size:11px;font-weight:700;padding:2px 7px;border-radius:4px;">%d %s</span>`, item.DaysLeft, daysLeft)
					case "HIGH":
						riskBadge = fmt.Sprintf(`<span style="display:inline-block;background:#fefce8;color:#ca8a04;font-size:11px;font-weight:700;padding:2px 7px;border-radius:4px;">%d %s</span>`, item.DaysLeft, daysLeft)
					}
					dayText = fmt.Sprintf("%d %s", item.DaysLeft, daysLeft)
				}
				_ = dayText

				fmt.Fprintf(&bodyHTML, `
        <tr>
          <td style="padding:12px 16px;background:%s;%s">
            <table role="presentation" width="100%%" cellpadding="0" cellspacing="0">
              <tr>
                <td style="font-size:13px;font-weight:600;color:#1e293b;">%d. %s</td>
                <td align="right">%s</td>
              </tr>
              <tr>
                <td style="font-size:12px;color:#64748b;padding-top:2px;">%d %s</td>
                <td align="right" style="font-size:12px;color:#64748b;padding-top:2px;">€%.2f</td>
              </tr>
            </table>
          </td>
        </tr>`, bg, borderBottom, i+1, item.ProductName, riskBadge, item.Quantity, units, item.Loss)
			}

			bodyHTML.WriteString(`
      </table>`)
		}

		// CTA button
		fmt.Fprintf(&bodyHTML, `
      <div style="text-align:center;margin-top:28px;">
        <a href="%s" style="display:inline-block;background:#059669;color:#ffffff;text-decoration:none;font-size:15px;font-weight:600;padding:14px 32px;border-radius:8px;letter-spacing:0.2px;">%s</a>
      </div>`, alertsURL, viewBtn)
	}

	// Sign-off
	fmt.Fprintf(&bodyHTML, `
      <p style="margin:28px 0 0;font-size:14px;color:#64748b;">%s<br><strong style="color:#059669;">%s</strong></p>`,
		haveAGreatDay, teamSig)

	// ── assemble full HTML ─────────────────────────────────────────
	html = fmt.Sprintf(`<!DOCTYPE html>
<html lang="%s">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width,initial-scale=1.0">
  <title>PharmaSense Daily Alert</title>
</head>
<body style="margin:0;padding:0;background-color:#f1f5f9;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;">
  <table role="presentation" width="100%%" cellpadding="0" cellspacing="0">
    <tr>
      <td align="center" style="padding:32px 12px;">
        <table role="presentation" width="100%%" style="max-width:600px;">

          <!-- Header -->
          <tr>
            <td style="background:#059669;border-radius:12px 12px 0 0;padding:22px 32px;">
              <p style="margin:0;font-size:20px;font-weight:800;color:#ffffff;letter-spacing:-0.5px;">💊 PharmaSense</p>
              <p style="margin:3px 0 0;font-size:13px;color:#a7f3d0;">Daily Inventory Alert</p>
            </td>
          </tr>

          <!-- Body -->
          <tr>
            <td style="background:#ffffff;padding:28px 32px;">
              %s
            </td>
          </tr>

          <!-- Footer -->
          <tr>
            <td style="background:#f8fafc;border-radius:0 0 12px 12px;padding:14px 32px;border-top:1px solid #e2e8f0;">
              <p style="margin:0;font-size:12px;color:#94a3b8;text-align:center;">
                PharmaSense &middot; <a href="%s" style="color:#059669;text-decoration:none;">%s</a>
              </p>
            </td>
          </tr>

        </table>
      </td>
    </tr>
  </table>
</body>
</html>`,
		d.Language,
		bodyHTML.String(),
		settingsURL, managePrefs,
	)

	return subject, html
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
