#!/bin/sh

if [ -z "$LTE_DIR" ]; then
  echo "LTE_DIR environment not set"
  exit 1
fi

if [ -z "$LTE_VERSION" ]; then
  echo "LTE_VERSION environment not set"
  exit 1
fi

sudo docker build -t lighttransport/lte_builder .

ABS_SH=`readlink -f $0`
ABS_DIR=`dirname $ABS_SH`

sudo -E rm -rf $ABS_DIR/docker_dist
mkdir $ABS_DIR/docker_dist

sudo -E docker run -v $LTE_DIR:/tmp/lte -v $ABS_DIR:/tmp/worker -v $ABS_DIR/docker_dist:/tmp/docker_dist lighttransport/lte_builder /tmp/internal.sh $LTE_VERSION

sudo -E cp internal_dockerfile $ABS_DIR/docker_dist/Dockerfile
sudo -E cp -r ../demo/scene $ABS_DIR/docker_dist/

cd $ABS_DIR/docker_dist; sudo docker build -t lighttransport/lte_worker .

cd $ABS_DIR; sudo -E rm -rf $ABS_DIR/docker_dist
