FROM ubuntu

MAINTAINER peryaudo, peryaudo@gmail.com

RUN apt-get update && apt-get install -y wget git mercurial

RUN cd /tmp && wget --no-check-certificate https://storage.googleapis.com/golang/go1.2.2.linux-amd64.tar.gz
RUN cd /tmp && tar -C /usr/local -xzf go1.2.2.linux-amd64.tar.gz

ENV PATH /usr/local/go/bin:$PATH
ENV GOPATH /tmp/workspace

ADD rotate_demo.go /tmp/workspace/src/rotate_demo/rotate_demo.go
ADD index.html /tmp/index.html
RUN cd /tmp/workspace/src/rotate_demo/ && go get && go build && cp rotate_demo /bin/rotate_demo

