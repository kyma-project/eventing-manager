package eventing

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kcorev1 "k8s.io/api/core/v1"

	k8smocks "github.com/kyma-project/eventing-manager/pkg/k8s/mocks"
	"github.com/kyma-project/eventing-manager/test/utils"
)

func Test_readDomainFromConfigMap(t *testing.T) {
	// given
	ctx := context.TODO()

	cm := &kcorev1.ConfigMap{
		Data: map[string]string{
			shootInfoConfigMapKeyDomain: utils.Domain,
		},
	}

	kubeClient := func() *k8smocks.Client {
		kubeClient := new(k8smocks.Client)
		kubeClient.On("GetConfigMap", ctx, shootInfoConfigMapName, shootInfoConfigMapNamespace).
			Return(cm, nil).Once()
		return kubeClient
	}

	wantError := error(nil)
	wantDomain := utils.Domain

	// when
	r := &Reconciler{kubeClient: kubeClient()}
	gotDomain, gotError := r.readDomainFromConfigMap(ctx)

	// then
	assert.Equal(t, wantError, gotError)
	assert.Equal(t, wantDomain, gotDomain)
}

func Test_domainMissingError(t *testing.T) {
	// given
	const errorMessage = "some error"
	err := fmt.Errorf(errorMessage)

	// when
	err0 := domainMissingError(nil)
	err1 := domainMissingError(err)

	// then
	require.Error(t, err0)
	require.Error(t, err1)
	require.False(t, strings.Contains(strings.ToLower(err0.Error()), "nil"))
	require.True(t, strings.Contains(err1.Error(), errorMessage))
}
