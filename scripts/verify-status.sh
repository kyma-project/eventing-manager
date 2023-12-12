#!/usr/bin/env bash

echo "Checking status of POST Jobs for Eventing-Manager"

REF_NAME="${1:-"main"}"
TIMEOUT_TIME="${2:-120}"
INTERVAL_TIME="${3:-3}"

# Generate job Status URL
STATUS_URL="https://api.github.com/repos/kyma-project/eventing-manager/commits/${REF_NAME}/status"

# Retry function
function retry {
	local start_time=$(date +%s)
	# Get status result
	local statusresult=$(curl -L -H "Accept: application/vnd.github+json" -H "X-GitHub-Api-Version: 2022-11-28" ${STATUS_URL})
	# Get overall state
	local fullstatus=$(echo $statusresult | jq '.state' | tr -d '"')

	while [ "$fullstatus" == "pending" ]; do
		current_time=$(date +%s)
		elapsed_time=$((current_time - start_time))

		if [ $elapsed_time -ge $TIMEOUT_TIME ]; then
			echo "Timeout reached. Exiting."
			exit 1
		fi

		echo "Status is 'pending'. Retrying in $interval seconds..."
		sleep $INTERVAL_TIME
	done

	# Show overall state to user
	echo "$statusresult"

	if [ "$fullstatus" == "success" ]; then
		echo "Success!"
	elif [ "$fullstatus" == "failed" ]; then
		echo "Failure! Exiting with an error."
		exit 1
	else
		echo "Invalid result: $result"
		exit 1
	fi
}

# Call retry function
retry