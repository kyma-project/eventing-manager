package validator

import (
	"fmt"
	"strconv"

	"k8s.io/apimachinery/pkg/util/validation/field"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	"github.com/kyma-project/eventing-manager/pkg/ems/api/events/types"
)

//nolint:gochecknoglobals // these are required for testing
var (
	sourcePath       = field.NewPath("spec").Child("source")
	typesPath        = field.NewPath("spec").Child("types")
	configPath       = field.NewPath("spec").Child("config")
	sinkPath         = field.NewPath("spec").Child("sink")
	typeMatchingPath = field.NewPath("spec").Child("typeMatching")
	namespacePath    = field.NewPath("metadata").Child("namespace")

	emptyErrDetail          = "must not be empty"
	invalidURIErrDetail     = "must be valid as per RFC 3986"
	duplicateTypesErrDetail = "must not have duplicate types"
	lengthErrDetail         = "must not be of length zero"
	minSegmentErrDetail     = fmt.Sprintf("must have minimum %s segments", strconv.Itoa(minEventTypeSegments))
	invalidPrefixErrDetail  = fmt.Sprintf("must not have %s as type prefix", validPrefix)
	stringIntErrDetail      = fmt.Sprintf("%s must be a stringified int value", eventingv1alpha2.MaxInFlightMessages)

	invalidQosErrDetail          = fmt.Sprintf("must be a valid QoS value %s or %s", types.QosAtLeastOnce, types.QosAtMostOnce)
	invalidAuthTypeErrDetail     = fmt.Sprintf("must be a valid Auth Type value %s", types.AuthTypeClientCredentials)
	invalidGrantTypeErrDetail    = fmt.Sprintf("must be a valid Grant Type value %s", types.GrantTypeClientCredentials)
	invalidTypeMatchingErrDetail = fmt.Sprintf("must be a valid TypeMatching value %s or %s", eventingv1alpha2.TypeMatchingExact, eventingv1alpha2.TypeMatchingStandard)

	missingSchemeErrDetail     = "must have URL scheme 'http' or 'https'"
	suffixMissingErrDetail     = fmt.Sprintf("must have valid sink URL suffix %s", clusterLocalURLSuffix)
	subDomainsErrDetail        = fmt.Sprintf("must have sink URL with %d sub-domains: ", subdomainSegments)
	namespaceMismatchErrDetail = "must have the same namespace as the subscriber: "
)

func makeInvalidFieldError(path *field.Path, subName, detail string) *field.Error {
	return field.Invalid(path, subName, detail)
}
