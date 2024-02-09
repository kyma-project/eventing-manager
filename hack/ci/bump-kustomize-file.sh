#!/usr/bin/env bash

# This script bumps the tag for the eventing-manager image in the kustomization file.

# Error handling.
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

TAG=$1
KUSTOMIZATION_FILE=${2-"config/manager/kustomization.yaml"}

# Bump kustomization file
awk -v ntv="$TAG" '/newTag:/ {sub(/newTag:[[:space:]]*[^[:space:]]*/, "newTag: " ntv)} {print}' "$KUSTOMIZATION_FILE" >tmp_file && mv tmp_file "$KUSTOMIZATION_FILE"
