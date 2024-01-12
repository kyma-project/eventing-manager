#!/bin/bash

source "${PROJECT_ROOT}/scripts/gardener/aws.sh"

gardener::init

gardener::provision_cluster
