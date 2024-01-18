#!/usr/bin/env bash

source "$(pwd)/scripts/utils/log.sh"

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

# utils::generate_random_number generates a random number between provided range.
# Arguments
# $1 - minimum
# $2 - maximum
# Example: utils::generate_random_number <minimum> <maximum>
function utils::generate_random_number() {
  # NOTE: This method should only echo the result.
  if [ "$#" -ne 2 ]
  then
    log::error "Invalid arguments. Usage: utils::generate_random_number <minimum> <maximum>"
    exit 1
  fi

  local min=$1
  local max=$2
  echo $(awk -v seed=$RANDOM "BEGIN{srand(seed); print int(rand()*($max-$min+1))+$min}")
}
