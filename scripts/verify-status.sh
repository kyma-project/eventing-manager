#!/usr/bin/env bash

echo "Checking status of POST Jobs for Eventing-Manager"

REF_NAME="${1:-"main"}"
STATUS_URL="https://api.github.com/repos/kyma-project/eventing-manager/commits/${REF_NAME}/status"
fullstatus=`curl -L -H "Accept: application/vnd.github+json" -H "X-GitHub-Api-Version: 2022-11-28" ${STATUS_URL} | head -n 2 `

sleep 10
echo $fullstatus

if [[ "$fullstatus" == *"success"* ]]; then
  echo "All jobs succeeded"
else
  echo "Jobs failed or pending - Check Prow status"
  exit 1
fi