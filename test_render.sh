#!/bin/bash

#set -x

if [ ! -f hostfile ]; then
  echo "hostfile not found"
  exit 1
fi

MYHOST=`cat hostfile`
SESSIONID=1
curl -o teapot.jpg http://${MYHOST}/sessions/${SESSIONID}/renders -XPOST
