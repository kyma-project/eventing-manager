#!/usr/bin/env bash

# This script verifies, that the current branch name is 'release-x.y', where x and y are multi-digit integers.

# Get the current Git branch name.
current_branch=$(git rev-parse --abbrev-ref HEAD)

# Define the pattern.
pattern="^release-[0-9]+[0-9]*\.[0-9]+[0-9]*$"

# Check if the branch name matches the pattern.
if [[ $current_branch =~ $pattern ]]; then
	echo "the current branch ($current_branch) is a release branch"
	exit 0
else
	echo "error; the current branch ($current_branch) is not a release branch"
	exit 1
fi
