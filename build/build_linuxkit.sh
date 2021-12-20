#!/bin/bash

set -e

SCRIPTPATH="$( cd "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"

mkdir -p ${SCRIPTPATH}/src
function finish {
   rm -rf ${SCRIPTPATH}/src
}
trap finish EXIT

cp -R ${SCRIPTPATH}/../{apic,cmd,configs,internal,pkg,go.mod,go.sum} ${SCRIPTPATH}/src/
linuxkit pkg build -docker $SCRIPTPATH

