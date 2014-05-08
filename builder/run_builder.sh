#!/bin/sh
ABS_SH=`readlink -f $0`
ABS_DIR=`dirname $ABS_SH`

if [ -z "$LTE_DIR" ]; then
  echo "LTE_DIR environment not set"
  exit 1
fi

if [ -z "$LTE_VERSION" ]; then
  echo "LTE_VERSION environment not set"
  exit 1
fi

sudo -E rm -rf $ABS_DIR/docker_dist
mkdir $ABS_DIR/docker_dist

sudo -E docker run -v $LTE_DIR:/tmp/lte -v $ABS_DIR/../worker:/tmp/worker -v $ABS_DIR/docker_dist:/tmp/docker_dist lighttransport/lte_builder /tmp/run_internal.sh $LTE_VERSION

sudo -E cp dockerfile_internal $ABS_DIR/docker_dist/Dockerfile

cd $ABS_DIR/docker_dist; sudo docker build -t lighttransport/lte_bin .

cd $ABS_DIR; sudo -E rm -rf $ABS_DIR/docker_dist

sudo docker tag lighttransport/lte_bin localhost:5000/lte_bin
sudo docker push localhost:5000/lte_bin

