#!/usr/bin/env bash

echo "Checking status of POST Jobs for Eventing-Manager"

REF_NAME="${1:-"main"}"
# Generate job Status URL
STATUS_URL="https://api.github.com/repos/kyma-project/eventing-manager/commits/${REF_NAME}/status"

# Wait for job result 
sleep 10

# Get status result
statusresult=`curl -L -H "Accept: application/vnd.github+json" -H "X-GitHub-Api-Version: 2022-11-28" ${STATUS_URL}`
# Get overall state
fullstatus=$(echo $statusresult | jq  '.state' | tr -d '"')

# Show overall state to user
echo $fullstatus

# Check state
if [[ $fullstatus == "success" ]]; then
  # Job: success
  echo "All jobs succeeded"
else
  # Job: failed or pending
  echo "Jobs failed or pending - Check Prow status"
  # Show job(s) details
  echo $statusresult | jq '.statuses[]' 
  exit 1
fi