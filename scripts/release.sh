MODULE_VERSION=${PULL_BASE_REF} make render-manifest
echo "Generated eventing-manager.yaml:"
cat eventing-manager.yaml
MODULE_VERSION=${PULL_BASE_REF} make module-build