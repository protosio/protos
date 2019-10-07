#!/bin/bash

go test -v \
	./internal/task \
    ./internal/resource \
	./internal/provider \
	./internal/installer \
	./internal/app \
	./internal/capability
