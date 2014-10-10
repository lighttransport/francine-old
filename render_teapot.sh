#!/bin/bash

#set -x

if [ ! -f hostfile ]; then
  echo "hostfile not found"
  exit 1
fi

MYHOST=`cat hostfile`
SESSIONID=`curl http://${MYHOST}/sessions -XPOST -d \{\"InputJson\"\:\ \"scene/teapot_redis.json\"\} | sed -e "s/.*\"\([0-9]*\)\"}/\1/g"`
curl http://${MYHOST}/sessions -XPOST -d \{\"InputJson\"\:\ \"scene/teapot_redis.json\"\}
curl http://${MYHOST}/sessions/${SESSIONID}/resources/scene/teapot_redis.json -XPUT --data-binary @demo/scene/teapot_redis.json
curl http://${MYHOST}/sessions/${SESSIONID}/resources/scene/teapot_scene.json -XPUT --data-binary @demo/scene/teapot_scene.json
curl http://${MYHOST}/sessions/${SESSIONID}/resources/scene/teapot.material.json -XPUT --data-binary @demo/scene/teapot.material.json
curl http://${MYHOST}/sessions/${SESSIONID}/resources/scene/shaders.json -XPUT --data-binary @demo/scene/shaders.json
curl http://${MYHOST}/sessions/${SESSIONID}/resources/scene/teapot.mesh -XPUT --data-binary @demo/scene/teapot.mesh
curl http://${MYHOST}/sessions/${SESSIONID}/resources/shader.c -XPUT --data-binary @demo/scene/shader.c
curl http://${MYHOST}/sessions/${SESSIONID}/resources/shader.h -XPUT --data-binary @demo/scene/shader.h
curl http://${MYHOST}/sessions/${SESSIONID}/resources/procedural-noise.c -XPUT --data-binary @demo/scene/procedural-noise.c
curl http://${MYHOST}/sessions/${SESSIONID}/resources/light.h -XPUT --data-binary @demo/scene/light.h
curl -o teapot.jpg http://${MYHOST}/sessions/${SESSIONID}/renders -XPOST

