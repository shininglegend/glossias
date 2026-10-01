package billing

import (
	"strings"
	"testing"
	"time"

	"github.com/stripe/stripe-go/v86/webhook"
)

func TestVerifyWebhookSecret(t *testing.T) {
	g := &stripeGateway{webhookSecret: "whsec_test"}
	if err := g.verifyWebhookSecret(); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyWebhookSecretRequiresSecret(t *testing.T) {
	g := &stripeGateway{}
	err := g.verifyWebhookSecret()
	if err == nil || !strings.Contains(err.Error(), "STRIPE_WEBHOOK_SECRET") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseWebhookRejectsOtherSecret(t *testing.T) {
	g := &stripeGateway{webhookSecret: "whsec_server"}
	payload := []byte(`{"id":"evt_x","object":"event","type":"checkout.session.completed","data":{"object":{"id":"cs_x","metadata":{"clerk_user_id":"user_1","course_id":"3"}}}}`)
	signed := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
		Payload:   payload,
		Secret:    "whsec_other",
		Timestamp: time.Now(),
	})
	_, err := g.ParseWebhook(signed.Payload, signed.Header)
	if err == nil {
		t.Fatal("payload signed with a different secret should be rejected")
	}
}
