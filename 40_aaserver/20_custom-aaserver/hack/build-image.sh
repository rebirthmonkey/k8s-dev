#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

KUBE_ROOT=$(dirname "${BASH_SOURCE[0]}")/../../../../..
source "${KUBE_ROOT}/hack/lib/util.sh"

# Register function to be called on EXIT to remove generated binary.
function cleanup {
  rm "${KUBE_ROOT}/vendor/20_custom-aaserver/artifacts/simple-image/kube-sample-apiserver"
}
trap cleanup EXIT

pushd "${KUBE_ROOT}/vendor/20_custom-aaserver"
cp -v ../../../../_output/local/bin/linux/amd64/20_custom-aaserver ./artifacts/simple-image/kube-20_custom-aaserver
docker build -t kube-sample-apiserver:latest ./artifacts/simple-image
popd
