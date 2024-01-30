package eventtype

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrPrefixNotFound = errors.New("prefix not found")
	ErrInvalidFormat  = errors.New("invalid format")
)

// parse splits the event-type using the given prefix and returns the application name, event and version
// or an error if the event-type format is invalid.
// A valid even-type format should be: prefix.application.event.version or application.event.version
// where event should consist of at least two segments separated by "." (e.g. businessObject.operation).
// Constraint: the application segment in the input event-type should not contain ".".
func parse(eventType, prefix string) (string, string, string, error) {
	if !strings.HasPrefix(eventType, prefix) {
		return "", "", "", ErrPrefixNotFound
	}

	// remove the prefix
	eventType = strings.ReplaceAll(eventType, prefix, "")
	eventType = strings.TrimPrefix(eventType, ".")

	// make sure that the remaining string has at least 4 segments separated by "."
	// (e.g. application.businessObject.operation.version)
	const minSAPEventTypeLength = 4
	parts := strings.Split(eventType, ".")
	if len(parts) < minSAPEventTypeLength {
		return "", "", "", ErrInvalidFormat
	}

	// parse the event-type segments
	applicationName := parts[0]
	businessObject := strings.Join(parts[1:len(parts)-2], "") // combine segments
	operation := parts[len(parts)-2]
	version := parts[len(parts)-1]
	event := fmt.Sprintf("%s.%s", businessObject, operation)

	return applicationName, event, version, nil
}
