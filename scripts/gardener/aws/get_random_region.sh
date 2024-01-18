#!/usr/bin/env bash

# This script returns a random AWS region name for Gardener cluster.
# Usage: No parameters are required to run this script. Simply call this script to get the region name.

source "$(pwd)/scripts/utils/utils.sh"

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -E          # needs to be set if we want the ERR trap.
set -o pipefail # prevents errors in a pipeline from being masked.
set -o errexit  # exit immediately when a command fails.

# only lists the regions where machine type: `c1.xlarge` is available.
AWS_REGIONS=(
  "eu-west-1"
  "us-east-1"
  "us-west-1"
  "us-west-2"
  "sa-east-1"
  "ap-northeast-1"
)

## MAIN Logic
# NOTE: This script should only echo the result.
RAND_INDEX=$(utils::generate_random_number 0 $(( ${#AWS_REGIONS[@]} - 1 )))
# print the random region name
echo ${AWS_REGIONS[$RAND_INDEX]}
