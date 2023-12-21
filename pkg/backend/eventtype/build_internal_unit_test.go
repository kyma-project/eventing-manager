package eventtype

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuilder(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name                                                        string
		givenPrefix, givenApplicationName, givenEvent, givenVersion string
		wantEventType                                               string
	}{
		{
			name:        "prefix is empty",
			givenPrefix: "", givenApplicationName: "test.app-1", givenEvent: "order.created", givenVersion: "v1",
			wantEventType: "test.app-1.order.created.v1",
		},
		{
			name:        "prefix is not empty",
			givenPrefix: "prefix", givenApplicationName: "test.app-1", givenEvent: "order.created", givenVersion: "v1",
			wantEventType: "prefix.test.app-1.order.created.v1",
		},
	}
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			eventType := build(testcase.givenPrefix, testcase.givenApplicationName, testcase.givenEvent, testcase.givenVersion)
			assert.Equal(t, testcase.wantEventType, eventType)
		})
	}
}
