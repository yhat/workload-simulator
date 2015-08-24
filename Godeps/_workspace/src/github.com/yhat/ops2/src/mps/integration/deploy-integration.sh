#!/bin/bash

set -e

OPSDIR=$GOPATH/src/github.com/yhat/ops2/src/mps/integration

cd $OPSDIR

# Get username, key, and endpoint from cli
USERNAME=$1
APIKEY=$2
OPS_ENDPOINT=$3
TAG=${4:-"latest"}

for val in "$USERNAME" "$APIKEY" "$OPS_ENDPOINT"; do
    if [ "$val" == "" ]; then
        echo "usage: deploy-integration.sh [username] [apikey] [ops_endpoint] [tag (default 'latest')]"
        exit 2
    fi
done

# deploy python models
cd $OPSDIR/python
DIRS=$(find . -maxdepth 1 -mindepth 1 -type d)
for dir in $DIRS; do
    cd $dir
    echo 'cd to: ' $dir
    model=${PWD##*/}
    dockerimg=yhat/integration-python-$model
    echo 'using docker image: ' $dockerimg:$TAG
    docker run -it --rm --net=host -e USERNAME="$USERNAME" -e APIKEY="$APIKEY" -e OPS_ENDPOINT="$OPS_ENDPOINT" $dockerimg:$TAG
    cd -
done


# # Add R models
cd $OPSDIR/r
DIRS=$(find . -maxdepth 1 -mindepth 1 -type d)
for dir in $DIRS; do
    cd $dir
    model=${PWD##*/}
    dockerimg=yhat/integration-r-$model
    docker run -it --rm --net=host -e USERNAME="$USERNAME" -e APIKEY="$APIKEY" -e OPS_ENDPOINT="$OPS_ENDPOINT" $dockerimg
    cd -
done
