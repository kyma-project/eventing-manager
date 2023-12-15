#!/usr/bin/env bash

# This script verifies, that the current branch name starts with 'release-'

CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "$CURRENT_BRANCH" == release-* ]]; then
	echo "Branch name starts with 'release-'."
else
	echo "Branch name does not start with 'release-'."
	exit 1
fi
