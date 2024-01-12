
source "${PROJECT_ROOT}/scripts/utils/log.sh"

# utils::check_required_vars checks if all provided variables are initialized
# Arguments
# $1 - list of variables
function utils::check_required_vars() {
  log::info "Checks if all provided variables are initialized"
  local discoverUnsetVar=false
  for var in "$@"; do
    if [ -z "${!var}" ] ; then
      log::warn "ERROR: $var is not set"
      discoverUnsetVar=true
    fi
  done
  if [ "${discoverUnsetVar}" = true ] ; then
    exit 1
  fi
}
