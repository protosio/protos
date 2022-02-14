#!/bin/bash

set -e

SCRIPTPATH="$( cd "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
linuxkit pkg build -docker -build-yml linuxkit_build.yaml  $SCRIPTPATH/..

