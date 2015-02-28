.PHONY: default all build bundles binary

BIN_DIR := bundles

GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
NETRACK_IMAGE := netrack$(if $(GIT_BRANCH),:$(GIT_BRANCH))

NETRACK_MOUNT := -v "$(CURDIR):/go/src/github.com/netrack/netrack"
NETRACK_RUN_DOCKER := docker run --rm -it --privileged $(NETRACK_MOUNT) "$(NETRACK_IMAGE)"

default: binary

all: build

binary: build
	$(NETRACK_RUN_DOCKER) scripts/make.sh binary

test: build
	$(NETRACK_RUN_DOCKER) scripts/make.sh test

shell: build
	$(NETRACK_RUN_DOCKER) bash

build: bundles
	docker build -t "$(NETRACK_IMAGE)" .

bundles:
	mkdir -p bundles
