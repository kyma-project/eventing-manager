package common

import (
	"time"
)

func Retry(attempts int, interval time.Duration, fn func() error) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var err error
	for { //nolint:gosimple//There is no range here.
		select {
		case <-ticker.C:
			attempts--
			err = fn()
			if err == nil || attempts == 0 {
				return err
			}
		}
	}
}

func RetryGet[T any](attempts int, interval time.Duration, fn func() (*T, error)) (*T, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var err error
	var obj *T
	for { //nolint:gosimple//There is no range here.
		select {
		case <-ticker.C:
			attempts--
			obj, err = fn()
			if err == nil || attempts == 0 {
				return obj, err
			}
		}
	}
}
