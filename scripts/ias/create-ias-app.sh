#!/usr/bin/env bash

source "${PROJECT_ROOT}/scripts/utils/utils.sh"

ias::validate_and_default() {
    requiredVars=(
        TEST_EVENTING_AUTH_IAS_PASSWORD
        TEST_EVENTING_AUTH_IAS_USER
        TEST_EVENTING_AUTH_IAS_URL
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
        export DISPLAY_NAME="${USER}""${DATE}"
        echo "INFO: no display name given, using the default display name: $DISPLAY_NAME"
    fi
}

ias::create-ias-app() {
    secret_name=$(kubectl get secrets -n kyma-system eventing-webhook-auth --ignore-not-found --no-headers -o custom-columns=":metadata.name")
    if [[ "${secret_name}" == "eventing-webhook-auth" ]]; then
      echo "Skip creation because secret kyma-system/eventing-webhook-auth already exits"
      return
    fi

    # set the credentials
    local user="${TEST_EVENTING_AUTH_IAS_USER}:${TEST_EVENTING_AUTH_IAS_PASSWORD}"

    # set uuid
    local uuid=$(uuidgen)
    echo "INFO: the generated uuid for your application is: $uuid"

    # create application
    local location=$(curl "${TEST_EVENTING_AUTH_IAS_URL}/Applications/v1/" \
    --silent \
    --include \
    --header 'Content-Type: application/json' \
    --user "${user}" \
    --data '{
      "id": "'"${uuid}"'",
      "name": "'"${uuid}"'",
      "multiTenantApp": false,
      "schemas": [
        "urn:sap:identity:application:schemas:core:1.0",
        "urn:sap:identity:application:schemas:extension:sci:1.0:Authentication"
      ],
      "branding" : {
        "displayName": "'"${DISPLAY_NAME}"'"
      },
      "urn:sap:identity:application:schemas:extension:sci:1.0:Authentication": {
        "ssoType": "openIdConnect",
        "subjectNameIdentifier": "uid",
        "rememberMeExpirationTimeInMonths": 3,
        "passwordPolicy": "https://accounts.sap.com/policy/passwords/sap/web/1.1",
        "userAccess": {
          "type": "internal",
          "userAttributesForAccess": [
            {
              "userAttributeName": "firstName",
              "isRequired": false
            },
            {
              "userAttributeName": "lastName",
              "isRequired": true
            },
            {
              "userAttributeName": "mail",
              "isRequired": true
            }
          ]
        },
        "assertionAttributes": [
          {
            "assertionAttributeName": "first_name",
            "userAttributeName": "firstName"
          },
          {
            "assertionAttributeName": "last_name",
            "userAttributeName": "lastName"
          },
          {
            "assertionAttributeName": "mail",
            "userAttributeName": "mail"
          },
          {
            "assertionAttributeName": "user_uuid",
            "userAttributeName": "userUuid"
          },
          {
            "assertionAttributeName": "locale",
            "userAttributeName": "language"
          }
        ],
        "spnegoEnabled": false,
        "biometricAuthenticationEnabled": false,
        "verifyMail": true,
        "forceAuthentication": false,
        "trustAllCorporateIdentityProviders": false,
        "allowIasUsers": false,
        "riskBasedAuthentication": {
          "defaultAction": [
            "allow"
          ]
        },
        "saml2Configuration": {
          "defaultNameIdFormat": "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified",
          "signSLOMessages": true,
          "requireSignedSLOMessages": true,
          "requireSignedAuthnRequest": false,
          "signAssertions": true,
          "signAuthnResponses": false,
          "responseElementsToEncrypt": "none",
          "digestAlgorithm": "sha256",
          "proxyAuthnRequest": {
            "authenticationContext": "none"
          }
        },
        "openIdConnectConfiguration": {}
      }
    }
    ' | grep -i ^Location: | cut -d: -f2- | sed 's/^ *\(.*\).*/\1/' | tr -d '[:space:]')

    exit_status=$?
    if [ ${exit_status} -ne 0 ]; then
      echo "ERROR: Error occurred while creating the application"
      exit 1
    fi

    echo "INFO: The following variable is needed to delete the application. Please save it:"
    echo "LOCATION=$location"
    echo "${location}" > ~/.ias_location

    # get client id
    client_id=$(curl --silent "${TEST_EVENTING_AUTH_IAS_URL}${location}" \
    --user "${user}" | jq -r '.["urn:sap:identity:application:schemas:extension:sci:1.0:Authentication"].clientId')

    # generate client secret
    client_secret=$(curl --silent "${TEST_EVENTING_AUTH_IAS_URL}${location}/apiSecrets" \
    --header 'Content-Type: application/json' \
    --user "${user}" \
    --data '{
      "authorizationScopes": [
        "oAuth",
        "manageApp",
        "manageUsers"
      ],
      "validTo": "2030-01-01T10:00:00Z"
    }' | jq -r '.secret')

    # generate token and certs url
    token_url="${TEST_EVENTING_AUTH_IAS_URL}/oauth2/token"
    certs_url="${TEST_EVENTING_AUTH_IAS_URL}/oauth2/certs"

    # create eventing webhook auth secret
    kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
type: Opaque
metadata:
  namespace: kyma-system
  name: eventing-webhook-auth
stringData:
  certs_url: "${certs_url}"
  token_url: "${token_url}"
  client_id: "${client_id}"
  client_secret: "${client_secret}"
---
EOF
}

## MAIN Logic

ias::validate_and_default

ias::create-ias-app
