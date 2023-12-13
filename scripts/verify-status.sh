#!/usr/bin/env bash

echo "Checking status of POST Jobs for Eventing-Manager"

REF_NAME="${1:-"main"}"
TIMEOUT_TIME="${2:-240}"
INTERVAL_TIME="${3:-3}"

# Generate job Status URL
STATUS_URL="https://api.github.com/repos/kyma-project/eventing-manager/commits/${REF_NAME}/status"

# Dates
START_TIME=$(date +%s)
TODAY_DATE=$(date '+%Y-%m-%d')

# Retry function
function retry {

	# Get status result
	local statusresult=$(curl -L -H "Accept: application/vnd.github+json" -H "X-GitHub-Api-Version: 2022-11-28" ${STATUS_URL})
	# Get overall state
	fullstatus=$(echo $statusresult | jq '.state' | tr -d '"')
	local tudayprowrunstate=$(echo $statusresult | jq --arg t $TODAY_DATE '.statuses[]|select(.created_at | contains($t))' | jq '.state' | tr -d '"')
	echo $statusresult | jq --arg t $TODAY_DATE '.statuses[]|select(.created_at | contains($t))' | jq '[.]'

	local CURRENT_TIME=$(date +%s)
	local elapsed_time=$((CURRENT_TIME - START_TIME))

	if [ $elapsed_time -ge $TIMEOUT_TIME ]; then
		echo "Timeout reached. Exiting."
		exit 1
	fi

	if [ "$fullstatus" == "success" ]; then
		echo "Success!"
	elif [ "$fullstatus" == "failed" ]; then
		# Show overall state to user
		echo "$statusresult"
		echo "Failure! Exiting with an error."
		exit 1
	elif [ "$fullstatus" == "pending" ]; then
		echo "Status is '$fullstatus'. Retrying in $INTERVAL_TIME seconds..."
		sleep $INTERVAL_TIME
	else
		echo "Invalid result: $result"

		exit 1
	fi

}

# Initial wait
sleep 10
# Call retry function
retry
while [ "$fullstatus" == "pending" ]; do
	retry
done
