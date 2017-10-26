#!/usr/bin/env bash

set -e

# docker run --rm -v `pwd`:/go/src/$package -w /go/src/$package -e BINARY=cni/veth golang:1.6.2 bash -c /go/src/$package/hack/build-single.sh
# we are in the project root dir
cur_dir=`pwd`
bin_prefix="galaxy"
package=git.code.oa.com/gaiastack/galaxy

mkdir -p go/src/`dirname $package`
ln -sfn $cur_dir $cur_dir/go/src/$package
export GOPATH=$cur_dir/go
function cleanup() {
	rm $cur_dir/go/src/$package
}
trap cleanup EXIT

go build -o bin/binary -v $package/$BINARY