package rbac

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	krbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/yaml"
)

// rbacDir returns the absolute path to the config/rbac directory.
func rbacDir() string {
	var pcs [1]uintptr
	runtime.Callers(1, pcs[:])
	frame, _ := runtime.CallersFrames(pcs[:]).Next()
	return filepath.Dir(frame.File)
}

func loadClusterRole(t *testing.T, fileName string) krbacv1.ClusterRole {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(rbacDir(), fileName))
	require.NoError(t, err)

	var cr krbacv1.ClusterRole
	require.NoError(t, yaml.Unmarshal(data, &cr))
	return cr
}

func TestViewClusterRole(t *testing.T) {
	clusterRole := loadClusterRole(t, "kyma_eventing_view_role.yaml")

	t.Run("should have correct name", func(t *testing.T) {
		assert.Equal(t, "kyma-eventing-view", clusterRole.Name)
	})

	t.Run("should have aggregate-to-view label", func(t *testing.T) {
		assert.Equal(t, "true", clusterRole.Labels["rbac.authorization.k8s.io/aggregate-to-view"])
	})

	t.Run("should contain only read verbs", func(t *testing.T) {
		allowedVerbs := map[string]struct{}{
			"get":   {},
			"list":  {},
			"watch": {},
		}
		for _, rule := range clusterRole.Rules {
			for _, verb := range rule.Verbs {
				_, ok := allowedVerbs[verb]
				assert.True(t, ok, "view role contains disallowed verb %q for resources %v", verb, rule.Resources)
			}
		}
	})

	t.Run("should cover both module API groups", func(t *testing.T) {
		groups := collectAPIGroups(clusterRole.Rules)
		assert.Contains(t, groups, "operator.kyma-project.io")
		assert.Contains(t, groups, "eventing.kyma-project.io")
	})

	t.Run("should not grant access to secrets", func(t *testing.T) {
		for _, rule := range clusterRole.Rules {
			for _, res := range rule.Resources {
				assert.NotEqual(t, "secrets", res, "view role must not grant access to secrets")
			}
		}
	})
}

func TestEditClusterRole(t *testing.T) {
	clusterRole := loadClusterRole(t, "kyma_eventing_edit_role.yaml")

	t.Run("should have correct name", func(t *testing.T) {
		assert.Equal(t, "kyma-eventing-edit", clusterRole.Name)
	})

	t.Run("should have aggregate-to-edit label", func(t *testing.T) {
		assert.Equal(t, "true", clusterRole.Labels["rbac.authorization.k8s.io/aggregate-to-edit"])
	})

	t.Run("should cover both module API groups", func(t *testing.T) {
		groups := collectAPIGroups(clusterRole.Rules)
		assert.Contains(t, groups, "operator.kyma-project.io")
		assert.Contains(t, groups, "eventing.kyma-project.io")
	})

	t.Run("should contain write verbs for main resources", func(t *testing.T) {
		requiredVerbs := []string{"create", "update", "patch", "delete", "deletecollection"}
		verbs := collectVerbsForMainResources(clusterRole.Rules)
		for _, v := range requiredVerbs {
			assert.Contains(t, verbs, v, "edit role is missing required verb %q", v)
		}
	})

	t.Run("should contain read verbs for main resources", func(t *testing.T) {
		verbs := collectVerbsForMainResources(clusterRole.Rules)
		for _, v := range []string{"get", "list", "watch"} {
			assert.Contains(t, verbs, v, "edit role is missing required verb %q", v)
		}
	})
}

// collectAPIGroups returns a deduplicated set of API groups from the given rules.
func collectAPIGroups(rules []krbacv1.PolicyRule) map[string]struct{} {
	groups := make(map[string]struct{})
	for _, rule := range rules {
		for _, g := range rule.APIGroups {
			groups[g] = struct{}{}
		}
	}
	return groups
}

// collectVerbsForMainResources returns a deduplicated set of verbs from rules
// that target main resources (excluding /status sub-resources).
func collectVerbsForMainResources(rules []krbacv1.PolicyRule) map[string]struct{} {
	verbs := make(map[string]struct{})
	for _, rule := range rules {
		hasMainResource := false
		for _, res := range rule.Resources {
			if res == "eventings" || res == "subscriptions" {
				hasMainResource = true
				break
			}
		}
		if hasMainResource {
			for _, v := range rule.Verbs {
				verbs[v] = struct{}{}
			}
		}
	}
	return verbs
}
