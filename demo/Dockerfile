FROM ubuntu

MAINTAINER peryaudo, peryaudo@gmail.com

EXPOSE 7000

RUN apt-get update && apt-get install -y wget git python-software-properties xz-utils
RUN sudo add-apt-repository ppa:chris-lea/node.js && sudo apt-get update && sudo apt-get install -y nodejs

RUN cd /tmp && wget http://llvm.org/releases/3.4/clang+llvm-3.4-x86_64-linux-gnu-ubuntu-13.10.tar.xz
RUN cd /tmp && tar -C /tmp -xJf clang+llvm-3.4-x86_64-linux-gnu-ubuntu-13.10.tar.xz && cp -r /tmp/clang+llvm-3.4-x86_64-linux-gnu-ubuntu-13.10/* /usr/local && rm -rf /tmp/clang+llvm-3.4-x86_64-linux-gnu-ubuntu-13.10

ADD javascripts /tmp/server/javascripts
ADD css /tmp/server/css
ADD data /tmp/server/data
ADD manual /tmp/server/manual
ADD package.json /tmp/server/package.json
RUN cd /tmp/server && npm config set registry http://registry.npmjs.org/ && npm install
ADD server.js /tmp/server/server.js
ADD index.html /tmp/server/index.html
ADD scene/shader.h /tmp/server/shader.h
ADD scene/procedural-noise.c /tmp/server/procedural-noise.c
ADD scene/light.h /tmp/server/light.h

