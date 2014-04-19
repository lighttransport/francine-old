#!/bin/sh
ABS_SH=`readlink -f $0`
ABS_DIR=`dirname $ABS_SH`

sudo -E rm -rf $ABS_DIR/docker_dist
mkdir $ABS_DIR/docker_dist

sudo -E docker run -v $LTE_DIR:/tmp/lte -v $ABS_DIR/../worker:/tmp/worker -v $ABS_DIR/docker_dist:/tmp/docker_dist lighttransport/lte_builder /tmp/run_internal.sh

sudo -E cp dockerfile_internal $ABS_DIR/docker_dist/Dockerfile

cd $ABS_DIR/docker_dist; sudo docker build -t lighttransport/lte_bin .

cd $ABS_DIR; sudo -E rm -rf $ABS_DIR/docker_dist

CONTAINER_ID=`sudo docker run -d lighttransport/lte_bin ls`

sudo -E docker export $CONTAINER_ID | gzip -c > $ABS_DIR/lighttransport-lte_bin.tar.gz
