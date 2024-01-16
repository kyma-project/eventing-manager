#!/usr/bin/env bash

source "${PROJECT_ROOT}/scripts/utils/utils.sh"

ias::validate_and_default() {
    requiredVars=(
        TEST_EVENTING_AUTH_IAS_PASSWORD
        TEST_EVENTING_AUTH_IAS_USER
        TEST_EVENTING_AUTH_IAS_URL
        IAS_APPLICATION_LOCATION
    )
    utils::check_required_vars "${requiredVars[@]}"

    # validations
    if [ "${#CLUSTER_NAME}" -gt 9 ]; then
        log::error "Provided cluster name is too long"
        return 1
    fi

    # set default values
    if [[ ! -n ${DISPLAY_NAME} ]]; then
        DATE=$(date +'-%d-%m-%Y-%T')
        DISPLAY_NAME="${USER}""${DATE}"
        echo "INFO: no display name given, using the default display name: $DISPLAY_NAME"
    fi
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
