#!/bin/bash

# builds full image path given dev and prod image repos and tag. If tag starts with PR, then it's a dev image.

DOCKER_IMAGE_REPO=europe-docker.pkg.dev/kyma-project/prod/eventing-manager
DOCKER_IMAGE_REPO_DEV=europe-docker.pkg.dev/kyma-project/dev/eventing-manager

image_tag=$1

if [[ $image_tag == PR* ]]; then
    full_image_path=$DOCKER_IMAGE_REPO_DEV:$image_tag
else
    full_image_path=$DOCKER_IMAGE_REPO:$image_tag
fi

echo $full_image_path
