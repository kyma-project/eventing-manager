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

// SetStatus sets the SubscriptionStatus of the stored Subscription under lock.
// It reports whether a Subscription existed for the key.
func (s *SafeSubscriptions) SetStatus(key string, status emstypes.SubscriptionStatus) bool {
	s.Lock()
	defer s.Unlock()
	subscription, ok := s.subscriptions[key]
	if !ok {
		return false
	}
	subscription.SubscriptionStatus = status
	return true
}

// SetWebhookAuth sets the WebhookAuth of the stored Subscription under lock.
// It reports whether a Subscription existed for the key.
func (s *SafeSubscriptions) SetWebhookAuth(key string, webhookAuth *emstypes.WebhookAuth) bool {
	s.Lock()
	defer s.Unlock()
	subscription, ok := s.subscriptions[key]
	if !ok {
		return false
	}
	subscription.WebhookAuth = webhookAuth
	return true
}

// GetSubscriptionCopy returns a copy of a stored Subscription under lock,
// so callers can read it without racing concurrent mutations.
func (s *SafeSubscriptions) GetSubscriptionCopy(key string) (emstypes.Subscription, bool) {
	s.RLock()
	defer s.RUnlock()
	subscription, ok := s.subscriptions[key]
	if !ok {
		return emstypes.Subscription{}, false
	}
	return *subscription, true
}
