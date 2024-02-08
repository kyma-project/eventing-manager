#!/usr/bin/env bash

VERSION=$1
KUSTOMIZATION_FILE=${2-"config/manager/kustomization.yaml"}

# Bump kustomization file
awk -v ntv="$VERSION" '/newTag:/ {sub(/newTag:[[:space:]]*[^[:space:]]*/, "newTag: " ntv)} {print}' "$KUSTOMIZATION_FILE" >tmp_file && mv tmp_file "$KUSTOMIZATION_FILE"
