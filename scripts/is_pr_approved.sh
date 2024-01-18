#!/usr/bin/env bash

# This script will check if a PR is approved or not.
# Usage: To run this script, provide the following environment variable(s) and run this script.
# - PR_URL - (example: https://github.com/kyma-project/eventing-manager/pull/123)

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -E          # needs to be set if we want the ERR trap.
set -o pipefail # prevents errors in a pipeline from being masked.
set -o errexit  # exit immediately when a command fails.

## MAIN Logic
echo "Checking approval status of : ${PR_URL}"
REVIEW_DECISION=$(gh pr view ${PR_URL} --json=reviewDecision | jq -r '.reviewDecision')

echo "REVIEW_DECISION: ${REVIEW_DECISION}"
if [[ ${REVIEW_DECISION} == "APPROVED"  ]]; then
  echo "Pull request is approved!"
  exit 0
fi

echo "Error: Pull request is not approved!"
exit 1
