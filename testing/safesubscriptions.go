package testing

import (
	"strings"
	"sync"

	emstypes "github.com/kyma-project/eventing-manager/pkg/ems/api/events/types"
)

// SafeSubscriptions encapsulates Subscriptions to provide mutual exclusion.
type SafeSubscriptions struct {
	sync.RWMutex
	subscriptions map[string]*emstypes.Subscription
}

// NewSafeSubscriptions returns a new instance of SafeSubscriptions.
func NewSafeSubscriptions() *SafeSubscriptions {
	return &SafeSubscriptions{
		sync.RWMutex{},
		make(map[string]*emstypes.Subscription),
	}
}

// GetSubscription returns a Subscription via the corresponding key.
func (s *SafeSubscriptions) GetSubscription(key string) *emstypes.Subscription {
	s.RLock()
	defer s.RUnlock()
	return s.subscriptions[key]
}

// DeleteSubscription deletes a Subscription via the corresponding key.
func (s *SafeSubscriptions) DeleteSubscription(key string) {
	s.Lock()
	defer s.Unlock()
	delete(s.subscriptions, key)
}

// DeleteSubscriptionsByName deletes all Subscriptions that contain the substring name in their own name.
func (s *SafeSubscriptions) DeleteSubscriptionsByName(name string) {
	s.Lock()
	defer s.Unlock()
	for k := range s.subscriptions {
		if strings.Contains(k, name) {
			delete(s.subscriptions, k)
		}
	}
}

// PutSubscription adds a Subscription and it's corresponding key to SafeSubscriptions.
func (s *SafeSubscriptions) PutSubscription(key string, subscription *emstypes.Subscription) {
	s.Lock()
	defer s.Unlock()
	s.subscriptions[key] = subscription
}
