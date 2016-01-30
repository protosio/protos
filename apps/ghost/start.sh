#!/bin/bash

SCRIPTPATH="$( cd "$(dirname "$0")" ; pwd -P )"

GHOST_DATA=$SCRIPTPATH/../../data/ghost

if [ ! -d "$GHOST_DATA" ]; then
  echo "--- cant find a data directory for Ghost, creating one"
  mkdir $GHOST_DATA
fi

docker run -d -p 8085:2368 -v $GHOST_DATA:/ghost-override dockerfile/ghost --name protos_ghost
