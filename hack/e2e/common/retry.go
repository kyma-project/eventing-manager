package common

import (
	"time"
)

func Retry(attempts int, interval time.Duration, fun func() error) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var err error
	for i := attempts; i > 0; i-- {
		<-ticker.C
		attempts--
		err = fun()
		if err == nil {
			return nil
		}
	}
	// Return nil if all attempts fail.
	return nil
}

func RetryGet[T any](attempts int, interval time.Duration, fun func() (*T, error)) (*T, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var err error
	var obj *T
	for i := attempts; i > 0; i-- {
		<-ticker.C
		attempts--
		obj, err = fun()
		if err == nil {
			return obj, nil
		}
	}
	// Return nil object if all attempts fail.
	return nil, err
}
