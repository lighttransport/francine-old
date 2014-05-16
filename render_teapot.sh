#!/bin/bash
curl http://localhost/sessions -XPOST -d \{\"InputJson\"\:\ \"scene/teapot_redis.json\"\}
curl http://localhost/sessions/1/resources/scene/teapot_redis.json -XPUT --data-binary @demo/scene/teapot_redis.json
curl http://localhost/sessions/1/resources/scene/teapot_scene.json -XPUT --data-binary @demo/scene/teapot_scene.json
curl http://localhost/sessions/1/resources/scene/teapot.material.json -XPUT --data-binary @demo/scene/teapot.material.json
curl http://localhost/sessions/1/resources/scene/shaders.json -XPUT --data-binary @demo/scene/shaders.json
curl http://localhost/sessions/1/resources/scene/teapot.mesh -XPUT --data-binary @demo/scene/teapot.mesh
curl http://localhost/sessions/1/resources/shader.c -XPUT --data-binary @demo/scene/shader.c
curl http://localhost/sessions/1/resources/shader.h -XPUT --data-binary @demo/scene/shader.h
curl http://localhost/sessions/1/resources/procedural-noise.c -XPUT --data-binary @demo/scene/procedural-noise.c
curl http://localhost/sessions/1/resources/light.h -XPUT --data-binary @demo/scene/light.h
curl -O http://localhost/sessions/1/renders -XPOST

