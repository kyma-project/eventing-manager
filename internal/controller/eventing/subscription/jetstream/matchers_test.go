package jetstream_test

import (
	"fmt"

	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	"github.com/kyma-project/eventing-manager/pkg/backend/jetstream"
	"github.com/kyma-project/eventing-manager/pkg/env"
)

func BeValidSubscription() gomegatypes.GomegaMatcher {
	return gomega.WithTransform(func(subscriber jetstream.Subscriber) bool {
		return subscriber.IsValid()
	}, gomega.BeTrue())
}

func BeNatsSubWithMaxPending(expectedMaxAckPending int) gomegatypes.GomegaMatcher {
	return gomega.WithTransform(func(subscriber jetstream.Subscriber) (int, error) {
		info, err := subscriber.ConsumerInfo()
		if err != nil {
			return -1, err
		}
		return info.Config.MaxAckPending, nil
	}, gomega.Equal(expectedMaxAckPending))
}

func BeJetStreamSubscriptionWithSubject(source, subject string,
	typeMatching eventingv1alpha2.TypeMatching, natsConfig env.NATSConfig,
) gomegatypes.GomegaMatcher {
	return gomega.WithTransform(func(subscriber jetstream.Subscriber) (bool, error) {
		info, err := subscriber.ConsumerInfo()
		if err != nil {
			return false, err
		}
		stream := jetstream.JetStream{
			Config: natsConfig,
		}
		result := info.Config.FilterSubject == stream.GetJetStreamSubject(source, subject, typeMatching)
		if !result {
			//nolint:goerr113 // no production code, but test helper functionality
			return false, fmt.Errorf(
				"BeJetStreamSubscriptionWithSubject expected %v to be equal to %v",
				info.Config.FilterSubject,
				stream.GetJetStreamSubject(source, subject, typeMatching),
			)
		}
		return true, nil
	}, gomega.BeTrue())
}

func BeExistingSubscription() gomegatypes.GomegaMatcher {
	return gomega.WithTransform(func(subscriber jetstream.Subscriber) bool {
		return subscriber != nil
	}, gomega.BeTrue())
}
