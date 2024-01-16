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

if [[ -z "${BUILD_JOB_NAME}" ]]; then
  echo "ERROR: BUILD_JOB_NAME is not set!"
  exit 1
fi

# set links for artifacts of pull requests.
ARTIFACTS_BASE_URL="https://gcsweb.build.kyma-project.io/gcs/kyma-prow-logs/pr-logs/pull/${BUILD_JOB_NAME}"
TEMPLATE_FILE_BASE_URL="${ARTIFACTS_BASE_URL}/${PR_NUMBER}/${BUILD_JOB_NAME}"
# if PR_NUMBER is not set, then set links for artifacts of main branch.
if [[ -z "${PR_NUMBER}" ]]; then
  ARTIFACTS_BASE_URL="https://gcsweb.build.kyma-project.io/gcs/kyma-prow-logs/logs/${BUILD_JOB_NAME}"
  TEMPLATE_FILE_BASE_URL="${ARTIFACTS_BASE_URL}"
fi

## Extract the prow job ID.
echo "Extracting prow job Id from: ${COMMIT_STATUS_JSON}"
TARGET_URL=$(echo ${COMMIT_STATUS_JSON} | jq -r '.target_url')
PROW_JOB_ID=$(echo ${TARGET_URL##*/})
echo "Prow Job ID: ${PROW_JOB_ID}, Link: ${TARGET_URL}"

## Download the build logs.
LOGS_FILE_NAME="${BUILD_JOB_NAME}-logs.txt"
LOGS_FILE_URL="${TEMPLATE_FILE_BASE_URL}/${PROW_JOB_ID}/build-log.txt"
echo "Downloading build logs from: ${LOGS_FILE_URL}"
curl -s -L -o ${LOGS_FILE_NAME}  ${LOGS_FILE_URL}

## extract the image name from build logs.
export IMAGE_NAME=$(cat ${LOGS_FILE_NAME} | grep "Successfully built image:" | awk -F " " '{print $NF}')
