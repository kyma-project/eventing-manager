#!/usr/bin/env bash

# Error handling.
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

RELEASE_TAG=$1

REPOSITORY=${REPOSITORY:-kyma-project/eventing-manager}
GITHUB_URL=https://api.github.com/repos/${REPOSITORY}
GITHUB_AUTH_HEADER="Authorization: token ${GH_TOKEN}"
CHANGELOG_FILE="CHANGELOG.md"

# The git describe --tag --abbrev=0 command is used to find the most recent tag that is reachable from a commit.
# The --tag option tells git describe to consider any tag found in the refs/tags namespace, enabling matching a lightweight (non-annotated) tag.
PREVIOUS_RELEASE=$(git describe --tags --abbrev=0)

# Generate the changelog in the CHANGELOG.md.
echo "## What has changed" >>${CHANGELOG_FILE}

# Iterate over all commits since the previous release.
git log "${PREVIOUS_RELEASE}"..HEAD --pretty=tformat:"%h" --reverse | while read -r commit; do
	# If the author of the commit is not kyma-bot, append the commit message to the changelog.
	COMMIT_AUTHOR=$(curl -H "${GITHUB_AUTH_HEADER}" -sS "${GITHUB_URL}/commits/${commit}" | jq -r '.author.login')
	if [ "${COMMIT_AUTHOR}" != "kyma-bot" ]; then
		git show -s "${commit}" --format="* %s by @${COMMIT_AUTHOR}" >>${CHANGELOG_FILE}
	fi
done

# Create a new contibutors file (with a unique name based on the process ID of the current shell).
NEW_CONTRIB=$$.authors

# Find unique authors who contributed since the last release, but not before it, and add them to the NEW_CONTRIB file.
join -v2 \
	<(curl -H "${GITHUB_AUTH_HEADER}" -sS "${GITHUB_URL}/compare/$(git rev-list --max-parents=0 HEAD)...${PREVIOUS_RELEASE}" | jq -r '.commits[].author.login' | sort -u) \
	<(curl -H "${GITHUB_AUTH_HEADER}" -sS "${GITHUB_URL}/compare/${PREVIOUS_RELEASE}...HEAD" | jq -r '.commits[].author.login' | sort -u) >${NEW_CONTRIB}

# Add new contributors to the 'new contributors' section of the changelog.
if [ -s ${NEW_CONTRIB} ]; then
	echo -e "\n## New contributors" >>${CHANGELOG_FILE}
	while read -r user; do
		REF_PR=$(grep "@${user}" ${CHANGELOG_FILE} | head -1 | grep -o " (#[0-9]\+)" || true)
		if [ -n "${REF_PR}" ]; then #reference found
			REF_PR=" in ${REF_PR}"
		fi
		echo "* @${user} made first contribution${REF_PR}" >>${CHANGELOG_FILE}
	done <${NEW_CONTRIB}
fi

# Append link to the full-changelog this changelog.
echo -e "\n**Full changelog**: https://github.com/$REPOSITORY/compare/${PREVIOUS_RELEASE}...${RELEASE_TAG}" >>${CHANGELOG_FILE}
