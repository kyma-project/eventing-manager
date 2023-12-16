#!/usr/bin/env bash

set -e

# This scrpit generates the sec-scanners-config by fetching all relevant images.

TAG=$1
OUTPUT_FILE=${2:-"sec-scanners-config.yaml"}
WEBHOOK_FILE=${3-"config/webhook/kustomization.yaml"}
PUBLISHER_FILE=${4-"config/manager/manager.yaml"}

# Fetch Webhook Image.
echo "fetching webhook image from ${WEBHOOK_FILE}"
WEBHOOK_IMAGE=$(cat "$WEBHOOK_FILE" | yq eval '.images[0].newName')
WEBHOOK_TAG=$(cat "$WEBHOOK_FILE" | yq eval '.images[0].newTag')
echo -e "webhook image is ${WEBHOOK_IMAGE}:${WEBHOOK_TAG} \n"

# Fetch Publisher Image.
echo "fetching publisher image from ${PUBLISHER_FILE}"
PUBLISHER_IMAGE=$(cat "$PUBLISHER_FILE" | yq eval '.spec.template.spec.containers[0].env[] | select(.name == "PUBLISHER_IMAGE") | .value')
echo -e "publisher image is ${PUBLISHER_IMAGE} \n"

# Generating File.
echo -e "generating to ${OUTPUT_FILE} \n"
cat <<EOF | tee ${OUTPUT_FILE}
# Dont edit this file, it is autogenerated by github action 'Create release'.
# The value for the publisher image are extracted from ${PUBLISHER_FILE}.
# The value for the webhook image are extracted from ${WEBHOOK_FILE}.yaml.
module-name: eventing
rc-tag: ${TAG}
protecode:
  - europe-docker.pkg.dev/kyma-project/prod/eventing-manager:${TAG}
  - ${PUBLISHER_IMAGE}
  - ${WEBHOOK_IMAGE}:${WEBHOOK_TAG}
whitesource:
  language: golang-mod
  subprojects: false
  exclude:
    - "**/test/**"
    - "**/*_test.go"
    - "/hack/**"
EOF
