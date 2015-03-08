# This file describes the standard way to build Netrack, using docker
#
# Usage:
#
# # Assemble the full dev environment. This is slow the first time.
# docker build .
#

FROM ubuntu:14.04
MAINTAINER Yasha Bubnov <girokompass@gmail.com> (@yashkin)

RUN sudo apt-get update
RUN sudo apt-get install -y git curl redis-server --no-install-recommends

RUN git config --global user.email 'netrack-dummy@example.com'

ENV GOVERSION 1.4.1
ENV GOPLATFORM linux-amd64

# Instal Go
RUN curl -skSL https://storage.googleapis.com/golang/go${GOVERSION}.${GOPLATFORM}.tar.gz | tar -xz -C /usr/local

ENV GOPATH /go
ENV GOROOT /usr/local/go
ENV PATH ${GOROOT}/bin:${GOPATH}/bin:${PATH}

RUN mkdir -p ${GOPATH}

# Fetch source tree
COPY . ${GOPATH}/src/github.com/netrack/netrack
WORKDIR go/src/github.com/netrack/netrack

ENV GIT_SSL_NO_VERIFY true

# Install dependency manager
RUN go get github.com/mattn/gom

# Fetch project dependencies
RUN gom install

# Export dependencies
ENV PATH $PWD/_vendor:$PATH
