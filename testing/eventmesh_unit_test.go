package testing_test

import (
	"net/url"
	"testing"

	. "github.com/onsi/gomega"

	testingutils "github.com/kyma-project/eventing-manager/testing"
)

func Test_GetRestAPIObject(t *testing.T) {
	g := NewGomegaWithT(t)

	urlString := "/messaging/events/subscriptions/my-subscription"
	urlObject, err := url.Parse(urlString)
	g.Expect(err).ShouldNot(HaveOccurred())
	restObject := testingutils.GetRestAPIObject(urlObject)
	g.Expect(restObject).To(Equal("my-subscription"))
}
