set -xe

# make sure the user has the necessary docker files to build this
# otherwise docker will attempt to download these unexpectedly
docker inspect continuumio/anaconda:latest > /dev/null
docker inspect yhat/scienceops-r:0.0.2 > /dev/null
docker inspect ubuntu:14.04 > /dev/null

for dir in `ls r`; do
    cd r/$dir
    docker build --force-rm=true -t yhat/integration-r-$dir .
    cd -
done

python generate-python-dockerfiles.py

for dir in `ls python`; do
    cd python/$dir
    docker build --force-rm=true -t "yhat/integration-python-$dir:latest" -f Dockerfile_latest .
    docker build --force-rm=true -t "yhat/integration-python-$dir:tip" -f Dockerfile_tip .
    cd -
done
