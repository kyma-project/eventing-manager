#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# Expected environment variables:
# PR_URL - Number of the PR with the changes to be merged

# wait until the PR is merged.
while ; do
  pr_state=$(gh pr view ${PR_URL} --json state --jq '.state')
  if [ "$pr_state" == "CLOSED" ]; then
    echo "ERROR! PR has been closed!"
    exit 1
  elif [ "$pr_state" == "MERGED" ]; then
    echo "PR has been merged!"
    exit 0
  fi
  echo "Waiting for ${PR_URL} to be merged"
  sleep 10
done
