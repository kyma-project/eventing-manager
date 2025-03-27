#!/usr/bin/env bash

# Error handling.
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# This scrpit generates the sec-scanners-config by fetching all relevant images.

TAG=$1
OUTPUT_FILE=${2:-"sec-scanners-config.yaml"}
PUBLISHER_FILE=${4-"config/manager/manager.yaml"}

# Fetch Publisher Image.
echo "fetching publisher image from ${PUBLISHER_FILE}"
PUBLISHER_IMAGE=$(yq eval '.spec.template.spec.containers[0].env[] | select(.name == "PUBLISHER_IMAGE") | .value' <"${PUBLISHER_FILE}")
echo -e "publisher image is ${PUBLISHER_IMAGE} \n"

# Generating File.
echo -e "generating to ${OUTPUT_FILE} \n"
cat <<EOF | tee "${OUTPUT_FILE}"
# The rc-tag (release candidate tag) marks the tag of the image that needs to be scanned before it can be released.
# The rc-tag will be added to this file by the github action release workflow with an automatic generated PR.
# Remove the rc-tag field after a successful release.
module-name: eventing
kind: kyma
rc-tag: ${TAG}
bdba:
  - europe-docker.pkg.dev/kyma-project/prod/eventing-manager:${TAG}
  - ${PUBLISHER_IMAGE}
mend:
  language: golang-mod
  exclude:
    - "**/test/**"
    - "**/*_test.go"
    - "/hack/**"
checkmarx-one:
  preset: go-default
  exclude:
    - "**/test/**"
    - "**/*_test.go"
    - "/hack/**"
EOF
