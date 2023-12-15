#!/usr/bin/env bash

# This script checks that the VERSION arg does follow the pattern x.y.z where x, y and z are integers.

VERSION="$1"

if [[ $VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	echo "Version format is valid"
else
	echo "Version format is invalid"
	exit 1
fi
