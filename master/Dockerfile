FROM ubuntu

MAINTAINER peryaudo, peryaudo@gmail.com

EXPOSE 6379 80

RUN apt-get update && apt-get install -y nodejs npm redis-server supervisor wget git

RUN cd /tmp && wget --no-check-certificate https://go.googlecode.com/files/go1.2.1.linux-amd64.tar.gz
RUN cd /tmp && tar -C /usr/local -xzf go1.2.1.linux-amd64.tar.gz

ENV PATH /usr/local/go/bin:$PATH
ENV GOPATH /tmp/workspace

RUN go get github.com/garyburd/redigo/redis

ADD server.js /tmp/server/server.js
ADD package.json /tmp/server/package.json
RUN cd /tmp/server && npm config set registry http://registry.npmjs.org/ && npm install

ADD supervisord.conf /etc/supervisor/conf.d/supervisord.conf
RUN mkdir -p /var/log/supervisor

ADD master.go /tmp/workspace/src/master/master.go

RUN cd /tmp/workspace/src/master/ && go build && cp master /bin/master
