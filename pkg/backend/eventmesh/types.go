package eventmesh

import (
	"strconv"
)

type HTTPStatusError struct {
	StatusCode int
}

func (e HTTPStatusError) Error() string {
	return strconv.Itoa(e.StatusCode)
}

func (e *HTTPStatusError) Is(target error) bool {
	t, ok := target.(*HTTPStatusError)
	if !ok {
		return false
	}
	return e.StatusCode == t.StatusCode
}

type OAuth2ClientCredentials struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	CertsURL     string
}

type Response struct {
	StatusCode int
	Error      error
}
