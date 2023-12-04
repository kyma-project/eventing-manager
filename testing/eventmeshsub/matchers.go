package eventmeshsub

import (
	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"

	emstypes "github.com/kyma-project/eventing-manager/pkg/ems/api/events/types"
)

func HaveWebhookAuth(webhookAuth emstypes.WebhookAuth) gomegatypes.GomegaMatcher {
	return gomega.WithTransform(func(s *emstypes.Subscription) emstypes.WebhookAuth {
		return *s.WebhookAuth
	}, gomega.Equal(webhookAuth))
}

func HaveEvents(events emstypes.Events) gomegatypes.GomegaMatcher {
	return gomega.WithTransform(func(s *emstypes.Subscription) emstypes.Events { return s.Events },
		gomega.Equal(events))
}

func HaveQoS(qos emstypes.Qos) gomegatypes.GomegaMatcher {
	return gomega.WithTransform(func(s *emstypes.Subscription) emstypes.Qos { return s.Qos },
		gomega.Equal(qos))
}

func HaveExemptHandshake(exemptHandshake bool) gomegatypes.GomegaMatcher {
	return gomega.WithTransform(func(s *emstypes.Subscription) bool { return s.ExemptHandshake },
		gomega.Equal(exemptHandshake))
}

func HaveWebhookURL(webhookURL string) gomegatypes.GomegaMatcher {
	return gomega.WithTransform(func(s *emstypes.Subscription) string { return s.WebhookURL },
		gomega.Equal(webhookURL))
}

func HaveStatusPaused() gomegatypes.GomegaMatcher {
	return gomega.WithTransform(func(s *emstypes.Subscription) emstypes.SubscriptionStatus {
		return s.SubscriptionStatus
	}, gomega.Equal(emstypes.SubscriptionStatusPaused))
}

func HaveStatusActive() gomegatypes.GomegaMatcher {
	return gomega.WithTransform(func(s *emstypes.Subscription) emstypes.SubscriptionStatus {
		return s.SubscriptionStatus
	}, gomega.Equal(emstypes.SubscriptionStatusActive))
}

func HaveContentMode(contentMode string) gomegatypes.GomegaMatcher {
	return gomega.WithTransform(func(s *emstypes.Subscription) string { return s.ContentMode },
		gomega.Equal(contentMode))
}

func HaveNonEmptyLastFailedDeliveryReason() gomegatypes.GomegaMatcher {
	return gomega.WithTransform(func(s *emstypes.Subscription) string {
		return s.LastFailedDeliveryReason
	}, gomega.Not(gomega.BeEmpty()))
}
