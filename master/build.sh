#!/bin/sh

sudo docker build -t lighttransport/lte_master .

CONTAINER_ID=`docker run -d lighttransport/lte_master ls`

docker export $CONTAINER_ID | gzip -c > lighttransport-lte_master.tar.gz
