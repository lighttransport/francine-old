FROM centos

MAINTAINER peryaudo, peryaudo@gmail.com

RUN yum install -y docker.io
RUN yum install -y tar git bzip2
RUN cd /tmp && curl -O https://storage.googleapis.com/golang/go1.2.2.linux-amd64.tar.gz
RUN cd /tmp && tar -C /usr/local -xzf go1.2.2.linux-amd64.tar.gz

RUN mkdir /tmp/workspace /tmp/worker /tmp/lte /tmp/docker_dist /tmp/workspace/src

ENV PATH /usr/local/go/bin:$PATH
ENV GOPATH /tmp/workspace

RUN go get github.com/garyburd/redigo/redis

ADD internal.sh /tmp/internal.sh
RUN chmod +x /tmp/internal.sh
