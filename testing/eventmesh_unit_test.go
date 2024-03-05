package testing_test

import (
	"net/url"
	"testing"

	eventingtesting "github.com/kyma-project/eventing-manager/testing"

	. "github.com/onsi/gomega"
)

func Test_GetRestAPIObject(t *testing.T) {
	gomega := NewGomegaWithT(t)

	urlString := "/messaging/events/subscriptions/my-subscription"
	urlObject, err := url.Parse(urlString)
	gomega.Expect(err).ShouldNot(HaveOccurred())
	restObject := eventingtesting.GetRestAPIObject(urlObject)
	gomega.Expect(restObject).To(Equal("my-subscription"))
}
