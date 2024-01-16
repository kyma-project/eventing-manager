#!/bin/bash

## This script requires the following env variables:
# COMMIT_STATUS_JSON (required, json)
# BUILD_JOB_NAME (required, string)
# PR_NUMBER (optional, int, If not set then will run for main branch. e.g. 82)

# Example of `COMMIT_STATUS_JSON`
# {
# "url": "https://api.github.com/repos/kyma-project/nats-manager/statuses/12345678765432345676543",
# "avatar_url": "https://avatars.githubusercontent.com/u/123456",
# "id": 123456789,
# "node_id": "SC_kwDOJBeAG123456789",
# "state": "success",
# "description": "Job succeeded.",
# "target_url": "https://status.build.kyma-project.io/view/gs/kyma-prow-logs/pr-logs/pull/kyma-project_nats-manager/81/pull-nats-module-build/123456789",
# "context": "pull-nats-module-build",
# "created_at": "2023-07-18T11:39:23Z",
# "updated_at": "2023-07-18T11:39:23Z"
# }

## check if required ENVs are provided.
if [[ -z "${COMMIT_STATUS_JSON}" ]]; then
  echo "ERROR: COMMIT_STATUS_JSON is not set!"
  exit 1
fi

## download the html of prow page.
curl -s -L -o prow-html.html $(echo ${COMMIT_STATUS_JSON} | jq -r '.target_url')

## define the log file url.
LOGS_FILE_URL="$(cat prow-html.html | grep ">Artifacts</a>" | awk -F'"' '{print $2}')/build-log.txt"

### Download the build logs.
echo "Downloading build logs from: ${LOGS_FILE_URL}"
curl -s -L -o build-log.txt  ${LOGS_FILE_URL}

## extract the image name from build logs.
export IMAGE_NAME=$(cat ${LOGS_FILE_NAME} | grep "Successfully built image:" | awk -F " " '{print $NF}')

## validate if image exists
echo "Validate that image: ${IMAGE_NAME} exists.."
docker pull ${IMAGE_NAME}

echo ${IMAGE_NAME} > image.name
