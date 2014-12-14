#!/bin/bash

docker run -ti -v /home/al3x/code/egor:/go/src/egor/ -v /var/run/docker.sock:/var/run/docker.sock golang:egor /bin/bash
