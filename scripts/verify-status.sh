#!/usr/bin/env bash

echo "Checking status of POST Jobs for Eventing-Manager"

REF_NAME="${1:-"main"}"
TIMEOUT_TIME="${2:-600}"
INTERVAL_TIME="${3:-3}"
INITIAL_WAIT_TIME="${4:-30}"

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

	# Collect latest run related data
	local latestrun=$(echo $statusresult | jq '.statuses[-1]')
	local latestrun_state=$(echo $latestrun | jq '.state' | tr -d '"')
	local latestrun_createdat=$(echo $latestrun | jq '.created_at' | tr -d '"')
	local latestrun_targeturl=$(echo $latestrun | jq '.target_url' | tr -d '"')

	# Check Today's run data
	if [[ $latestrun_createdat == *"$TODAY_DATE"* ]]; then
		echo $latestrun_createdat
		echo $latestrun_state
		echo $latestrun_targeturl
	fi

	# Show all execution for Today
	echo $statusresult | jq --arg t $TODAY_DATE '.statuses[]|select(.created_at | contains($t))'

	# Date time for time-out
	local CURRENT_TIME=$(date +%s)
	local elapsed_time=$((CURRENT_TIME - START_TIME))

	# Check time-out
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
sleep $INITIAL_WAIT_TIME
# Call retry function
retry
while [ "$fullstatus" == "pending" ]; do
	retry
done
