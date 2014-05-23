FROM ubuntu

MAINTAINER peryaudo, peryaudo@gmail.com

EXPOSE 7000

RUN apt-get update && apt-get install -y wget git software-properties-common python-software-properties xz-utils
RUN sudo add-apt-repository ppa:chris-lea/node.js && sudo apt-get update && sudo apt-get install -y nodejs

RUN cd /tmp && wget http://llvm.org/releases/3.4/clang+llvm-3.4-x86_64-linux-gnu-ubuntu-13.10.tar.xz
RUN cd /tmp && tar -C /tmp -xJf clang+llvm-3.4-x86_64-linux-gnu-ubuntu-13.10.tar.xz && cp -r /tmp/clang+llvm-3.4-x86_64-linux-gnu-ubuntu-13.10/* /usr/local && rm -rf /tmp/clang+llvm-3.4-x86_64-linux-gnu-ubuntu-13.10

ADD javascripts /tmp/server/javascripts
ADD css /tmp/server/css
ADD data /tmp/server/data
ADD manual /tmp/server/manual
ADD package.json /tmp/server/package.json
RUN cd /tmp/server && npm config set registry http://registry.npmjs.org/ && npm install
ADD index.html /tmp/server/index.html
ADD scene/teapot_redis.json /tmp/server/teapot_redis.json
ADD scene/teapot_scene.json /tmp/server/teapot_scene.json
ADD scene/teapot.material.json /tmp/server/teapot.material.json
ADD scene/shaders.json /tmp/server/shaders.json
ADD scene/teapot.mesh /tmp/server/teapot.mesh
ADD scene/shader.c /tmp/server/shader.c
ADD scene/shader.h /tmp/server/shader.h
ADD scene/procedural-noise.c /tmp/server/procedural-noise.c
ADD scene/light.h /tmp/server/light.h
ADD server.js /tmp/server/server.js
