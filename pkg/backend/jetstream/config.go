package jetstream

import "github.com/kyma-project/eventing-manager/pkg/env"

// Validate ensures that the NatsConfig is valid and therefore can be used safely.
//

func Validate(natsConfig env.NATSConfig) error {
	if natsConfig.JSStreamName == "" {
		return ErrEmptyStreamName
	}
	if len(natsConfig.JSStreamName) > jsMaxStreamNameLength {
		return ErrStreamNameTooLong
	}
	if _, err := toJetStreamStorageType(natsConfig.JSStreamStorageType); err != nil {
		return err
	}
	if _, err := toJetStreamRetentionPolicy(natsConfig.JSStreamRetentionPolicy); err != nil {
		return err
	}
	if _, err := toJetStreamDiscardPolicy(natsConfig.JSStreamDiscardPolicy); err != nil {
		return err
	}
	return nil
}
