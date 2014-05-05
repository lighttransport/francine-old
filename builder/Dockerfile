FROM centos

MAINTAINER peryaudo, peryaudo@gmail.com

RUN yum install -y tar git bzip2
RUN cd /tmp && curl -O https://go.googlecode.com/files/go1.2.1.linux-amd64.tar.gz
RUN cd /tmp && tar -C /usr/local -xzf go1.2.1.linux-amd64.tar.gz
RUN mkdir /tmp/workspace /tmp/worker /tmp/lte /tmp/docker_dist /tmp/workspace/src

ENV PATH /usr/local/go/bin:$PATH
ENV GOPATH /tmp/workspace

RUN go get github.com/garyburd/redigo/redis

ADD run_internal.sh /tmp/run_internal.sh
ADD scene /tmp/scene
