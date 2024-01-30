package utils

import (
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/pkg/errors"
	kcorev1 "k8s.io/api/core/v1"

	emerrors "github.com/kyma-project/eventing-manager/pkg/errors"
)

const randStringlength = 10

var ErrParseSink = errors.Errorf("failed to parse subscription sink URL")

// GetPortNumberFromURL converts string port from url.URL to uint32 port.
func GetPortNumberFromURL(u url.URL) (uint32, error) {
	const (
		httpPort  = 80
		httpsPort = 443
	)
	port := uint32(0)
	sinkPort := u.Port()
	if sinkPort != "" {
		u64, err := strconv.ParseUint(sinkPort, 10, 32)
		if err != nil {
			return port, errors.Wrapf(err, "convert port failed %s", u.Port())
		}
		port = uint32(u64)
	}
	if port == uint32(0) {
		switch strings.ToLower(u.Scheme) {
		case "https":
			port = uint32(httpsPort)
		default:
			port = uint32(httpPort)
		}
	}
	return port, nil
}

// ContainsString checks if a string is contained in a slice of strings.
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func RemoveString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return result
}

func BoolPtr(b bool) *bool {
	return &b
}

func Int32Ptr(i int32) *int32 {
	return &i
}

func Int64Ptr(i int64) *int64 {
	return &i
}

func StringPtr(s string) *string {
	return &s
}

func ProcMountTypePtr(p kcorev1.ProcMountType) *kcorev1.ProcMountType {
	return &p
}

// for Random string generation.
const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec,gochecknoglobals

// GetRandString returns a random string of the given length.
func GetRandString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// IsValidScheme returns true if the sink scheme is http or https, otherwise returns false.
func IsValidScheme(sink string) bool {
	return strings.HasPrefix(sink, "http://") || strings.HasPrefix(sink, "https://")
}

func GetSinkData(sink string) (string, []string, error) {
	sURL, err := url.ParseRequestURI(sink)
	if err != nil {
		return "", nil, emerrors.MakeError(ErrParseSink, err)
	}
	trimmedHost := strings.Split(sURL.Host, ":")[0]
	subDomains := strings.Split(trimmedHost, ".")
	return trimmedHost, subDomains, nil
}

func GetCloudEvent(eventType string) event.Event {
	if eventType == "" {
		eventType = GetRandString(randStringlength)
	}
	newEvent := cloudevents.NewEvent()
	newEvent.SetType(eventType)
	newEvent.SetID(GetRandString(randStringlength))
	return newEvent
}

// IsEmpty returns true if the given string is empty.
// If the given string consists of whitespaces only, it is treated as an empty string.
func IsEmpty(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}
