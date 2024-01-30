//nolint:goerr113 // no need to wrap errors in the test for the error package
package httpclient

import (
	"errors"
	"fmt"
	"net/url"
	"testing"
)

func TestErrorDescription(t *testing.T) {
	tableTests := []struct {
		name            string
		giveErr         error
		wantDescription string
	}{
		{
			name: "all arguments given",
			giveErr: func() error {
				wrapped := NewError(fmt.Errorf("my error"), WithStatusCode(500), WithMessage("this is the http response"))
				return wrapped
			}(),
			wantDescription: `message: this is the http response; status code: 500; cause: my error`,
		},
		{
			name: "cause only",
			giveErr: func() error {
				existing := url.Error{
					Op:  "Delete",
					URL: "/foo/bar",
					Err: errors.New("unsupported protocol scheme"),
				}
				wrapped := NewError(&existing)
				return wrapped
			}(),
			wantDescription: `cause: Delete "/foo/bar": unsupported protocol scheme`,
		},
		{
			name: "message only",
			giveErr: func() error {
				wrapped := NewError(nil, WithMessage("message"))
				return wrapped
			}(),
			wantDescription: `message: message`,
		},
		{
			name: "status code only",
			giveErr: func() error {
				wrapped := NewError(nil, WithStatusCode(200))
				return wrapped
			}(),
			wantDescription: `status code: 200`,
		},
	}

	for _, tt := range tableTests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.giveErr.Error() != tt.wantDescription {
				t.Errorf("error message should not be %q, but: %q", tt.giveErr.Error(), tt.wantDescription)
			}
		})
	}
}
