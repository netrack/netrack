.PHONY: default
.PHONY: binary database test
.PHONY: docker-build docker-binary docker-test docker-shell

BUNDLES            := bundles

GIT_BRANCH         := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
NETRACK_IMAGE      := netrack$(if $(GIT_BRANCH),:$(GIT_BRANCH))

NETRACK_MOUNT      := -v "$(CURDIR):/go/src/github.com/netrack/netrack"
NETRACK_RUN_DOCKER := docker run --rm -it --privileged $(NETRACK_MOUNT) "$(NETRACK_IMAGE)"

# Netrack packages
NETRACK_PKG        += .
NETRACK_PKG        += config config/environment
NETRACK_PKG        += controller
NETRACK_PKG        += database
NETRACK_PKG        += httputil
NETRACK_PKG        += httprest/format httprest/v1
NETRACK_PKG        += ioutil
NETRACK_PKG        += logging
NETRACK_PKG        += mechanism mechanism/injector mechanism/mechutil mechanism/rpc
NETRACK_PKG        += netutil/drivers netutil/ip.v4 netutil/ofp.v13

# Netrack source code
NETRACK_SRC        := $(wildcard $(addsuffix /*.go,$(NETRACK_PKG)))

# Netrack tests
#NETRACK_TEST       := $(addprefix test/,$(NETRACK_PKG))

default: binary

# Build executable outside of docker container, you will be
# warned about this, keep your development enviroment clean.
binary: bundles/netrack

# Run tests
test: $(NETRACK_SRC)
	scripts/make.sh test

# Drop and create a new database
database:
	scripts/make.sh database

# Rebuild binary if any source file was changed
bundles/netrack: $(BUNDLES) $(NETRACK_SRC)
	scripts/make.sh binary

docker-binary: docker-build
	$(NETRACK_RUN_DOCKER) scripts/make.sh binary

docker-test: docker-build
	$(NETRACK_RUN_DOCKER) scripts/make.sh test

docker-shell: docker-build
	$(NETRACK_RUN_DOCKER) bash

# Creates a new docker image
docker-build: $(BUNDLES)
	docker build -t "$(NETRACK_IMAGE)" .

# Create output directory
$(BUNDLES):
	mkdir -p bundles

# Remove created data
clean:
	rm -rf $(BUNDLES)
