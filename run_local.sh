#!/bin/bash

if [ -z $1 ]; then
	echo "please pass local ip address as the first argument"
	echo "compile master and worker image first"
	exit 1
fi

IP_ADDR=$1

trap 'sudo kill $(jobs -p)' SIGINT SIGTERM EXIT

sleep 5
sudo -E docker run -p 6379:6379 lighttransport/redis &
sleep 5
sudo -E docker run -p 80:80 -e REDIS_HOST=$IP_ADDR:6379 lighttransport/lte_master /bin/master &
sleep 5
sudo -E docker run -e REDIS_HOST=$IP_ADDR:6379 -e WORKER_NAME=lte-worker lighttransport/lte_worker /bin/worker &
sleep 5
sudo -E docker run -p 7000:7000 -e REST_HOST=$IP_ADDR lighttransport/lte_demo /usr/bin/nodejs /tmp/server/server.js

