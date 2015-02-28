#!/usr/bin/env bash
set -e

NETRACK_PKG='github.com/netrack/netrack'

if [ "$(pwd)" != "/go/src/${NETRACK_PKG}" ]; then
    echo "### WARN: I don't seem to be running in the Docker container"
fi

BUNDLES=( binary test )

if [ ${#@} -ne 0 ]; then
    echo $@
    BUNDLES=$@
fi

SCRIPTS="$(cd "$(dirname ${BASH_SOURCE[0]})" && pwd)"

bundle() {
	bundlescript="${SCRIPTS}/make/$1"
    if [ ! -x ${bundlescript} ]; then
        echo "### ERR: ${bundlescript} Not found"
        exit 1
    fi

    source ${bundlescript} "${PWD}/bundles"

}

VERSION=$(bundle version)
REVISION=$(bundle revision)

for bundle in ${BUNDLES[@]}; do
    echo "=== INFO: Making bundle: ${bundle}"
    bundle "${bundle}"
done
