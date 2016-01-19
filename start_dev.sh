#!/bin/bash

if [ -z "$SSH_AUTH_SOCK" ]; then
    echo "SSH AGENT socket not set"
else
    SSH_VOLUME="-v $SSH_AUTH_SOCK:/tmp/ssh_agent -e SSH_AUTH_SOCK=/tmp/ssh_agent"
fi

docker rm -f egor
docker run -ti -p 80:80 -v /egor:/go/src/egor/ $SSH_VOLUME -v /var/run/docker.sock:/var/run/docker.sock --name=egor golang:egor /bin/bash
