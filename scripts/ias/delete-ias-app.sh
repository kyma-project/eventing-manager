#!/usr/bin/env bash

# This script will delete the provided IAS application.
#
# Usage: To run this script, set the following environment variables and run this script.
# - TEST_EVENTING_AUTH_IAS_PASSWORD
# - TEST_EVENTING_AUTH_IAS_USER
# - TEST_EVENTING_AUTH_IAS_URL
# - IAS_APPLICATION_LOCATION

source "${PROJECT_ROOT}/scripts/utils/utils.sh"

ias::validate_and_default() {
    # validations
    requiredVars=(
        TEST_EVENTING_AUTH_IAS_PASSWORD
        TEST_EVENTING_AUTH_IAS_USER
        TEST_EVENTING_AUTH_IAS_URL
        IAS_APPLICATION_LOCATION
    )
    utils::check_required_vars "${requiredVars[@]}"
}

ias::delete-ias-app() {
    LOCATION="${IAS_APPLICATION_LOCATION}"

    # set the credentials
    local user="${TEST_EVENTING_AUTH_IAS_USER}:${TEST_EVENTING_AUTH_IAS_PASSWORD}"

    # delete application
    curl -s -o /dev/null -X DELETE "${TEST_EVENTING_AUTH_IAS_URL}${LOCATION}" --user "${user}"
    exit_code=$?
    if [ $exit_code -eq 0 ]; then
        echo "INFO: Delete was successful"
    else
        echo "ERROR: Delete was not successful. Check the IAS UI to delete the app manually."
    fi
}


## MAIN Logic

ias::validate_and_default

ias::delete-ias-app
