FROM ubuntu

MAINTAINER peryaudo, peryaudo@gmail.com

EXPOSE 6379 80

RUN apt-get update && apt-get install -y redis-server supervisor wget git python-software-properties clang
RUN sudo add-apt-repository ppa:chris-lea/node.js && sudo apt-get update && sudo apt-get install -y nodejs

RUN cd /tmp && wget --no-check-certificate https://go.googlecode.com/files/go1.2.1.linux-amd64.tar.gz
RUN cd /tmp && tar -C /usr/local -xzf go1.2.1.linux-amd64.tar.gz

ENV PATH /usr/local/go/bin:$PATH
ENV GOPATH /tmp/workspace

RUN apt-get install -y mercurial
RUN go get github.com/garyburd/redigo/redis
RUN go get code.google.com/p/goauth2/oauth

ADD supervisord.conf /etc/supervisor/conf.d/supervisord.conf
RUN mkdir -p /var/log/supervisor

ADD master.go /tmp/workspace/src/master/master.go
ADD cloud-config.yaml /tmp/cloud-config.yaml

RUN cd /tmp/workspace/src/master/ && go build && cp master /bin/master

ADD javascripts /tmp/server/javascripts
ADD css /tmp/server/css
ADD data /tmp/server/data
ADD manual /tmp/server/manual
ADD package.json /tmp/server/package.json
RUN cd /tmp/server && npm config set registry http://registry.npmjs.org/ && npm install
ADD server.js /tmp/server/server.js
ADD index.html /tmp/server/index.html
