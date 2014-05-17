FROM ubuntu

MAINTAINER peryaudo, peryaudo@gmail.com

EXPOSE 6379 80

RUN apt-get update && apt-get install -y wget git mercurial

RUN cd /tmp && wget --no-check-certificate https://storage.googleapis.com/golang/go1.2.2.linux-amd64.tar.gz
RUN cd /tmp && tar -C /usr/local -xzf go1.2.2.linux-amd64.tar.gz

ENV PATH /usr/local/go/bin:$PATH
ENV GOPATH /tmp/workspace

RUN go get github.com/garyburd/redigo/redis
RUN go get code.google.com/p/goauth2/oauth

ADD cloud-config-worker.yaml /tmp/cloud-config-worker.yaml

ADD master.go /tmp/workspace/src/master/master.go
ADD rest.go /tmp/workspace/src/master/rest.go
RUN cd /tmp/workspace/src/master/ && go build && cp master /bin/master

