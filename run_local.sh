#!/bin/bash

if [ -z $1 ]; then
	echo "please pass local ip address as the first argument"
	echo "compile master and worker image first"
	exit 1
fi

IP_ADDR=$1

trap 'sudo kill $(jobs -p)' SIGINT SIGTERM EXIT

etcd &
sleep 5
curl -L http://$IP_ADDR:4001/v2/keys/redis-server -XPUT -d value="$IP_ADDR:6379"
sudo -E docker run -p 6379:6379 dockerfile/redis &
sleep 5
sudo -E docker run -p 80:80 -e ETCD_HOST=$IP_ADDR:4001 lighttransport/lte_master /bin/master &
sleep 5
sudo -E docker run -e ETCD_HOST=$IP_ADDR:4001 -e WORKER_NAME=lte-worker lighttransport/lte_worker /bin/worker

