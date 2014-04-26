#!/bin/sh

PREV_CONTAINER_ID=`cat master_id 2> /dev/null`
sudo -E docker kill $PREV_CONTAINER_ID > /dev/null 2> /dev/null
sudo -E docker rm $PREV_CONTAINER_ID > /dev/null 2> /dev/null
sudo -E docker rmi `cat master_image_id 2> /dev/null` > /dev/null 2> /dev/null

PRIVATE_IPV4=`sudo printenv COREOS_PRIVATE_IPV4`

IMAGE_ID=`sudo -E docker import - < lighttransport-lte_master.tar.gz`
rm lighttransport-lte_master.tar.gz
echo $IMAGE_ID > master_image_id

sudo -E docker run -d -p 6379:6379 -p 80:80 -p 8080:8080 -v `pwd`/lte_bin:/tmp/lte_bin $IMAGE_ID /usr/bin/supervisord > master_id

etcdctl set /redis-server $PRIVATE_IPV4:6379
etcdctl set /worker-image http://$PRIVATE_IPV4:8080/lighttransport-lte_bin.tar.gz
