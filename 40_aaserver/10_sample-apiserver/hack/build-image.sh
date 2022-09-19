#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

KUBE_ROOT=$(dirname "${BASH_SOURCE[0]}")/../../../../..
source "${KUBE_ROOT}/hack/lib/util.sh"

# Register function to be called on EXIT to remove generated binary.
function cleanup {
  rm "${KUBE_ROOT}/vendor/10_sample-apiserver/artifacts/simple-image/kube-sample-apiserver"
}
trap cleanup EXIT

pushd "${KUBE_ROOT}/vendor/10_sample-apiserver"
cp -v ../../../../_output/local/bin/linux/amd64/10_sample-apiserver ./artifacts/simple-image/kube-10_sample-apiserver
docker build -t kube-sample-apiserver:latest ./artifacts/simple-image
popd
