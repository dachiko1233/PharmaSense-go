package notifications

import (
	"context"
	"fmt"
	"log/slog"

	openapi "github.com/twilio/twilio-go/rest/api/v2010"
	twilioClient "github.com/twilio/twilio-go"
)

type SMSService struct {
	client     *twilioClient.RestClient
	fromNumber string
	isMock     bool
}

func NewSMSService(accountSID, authToken, fromNumber string) *SMSService {
	isMock := accountSID == "" || len(accountSID) > 5 && accountSID[:5] == "mock_"
	if isMock {
		return &SMSService{fromNumber: fromNumber, isMock: true}
	}
	client := twilioClient.NewRestClientWithParams(twilioClient.ClientParams{
		Username: accountSID,
		Password: authToken,
	})
	return &SMSService{
		client:     client,
		fromNumber: fromNumber,
		isMock:     false,
	}
}

func (s *SMSService) Send(ctx context.Context, to, body string) error {
	if s.isMock {
		slog.Info("[MOCK SMS]",
			"to", to,
			"body", body,
		)
		return nil
	}

	params := &openapi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(s.fromNumber)
	params.SetBody(body)

	_, err := s.client.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("send sms: %w", err)
	}
	return nil
}

func (s *SMSService) SendCriticalAlert(ctx context.Context, to, pharmacyName, productName string, daysLeft int, estimatedLoss float64) error {
	body := fmt.Sprintf(
		"[PharmaSense] CRITICAL: %s — '%s' expires in %d days. Estimated loss: €%.2f. Log in to take action.",
		pharmacyName, productName, daysLeft, estimatedLoss,
	)
	return s.Send(ctx, to, body)
}
