#!/bin/sh

cd tmp

cp -R worker workspace/src

# copy LTE binary
cp lte/lte_linux_x64.1.1.2.tar.bz2 .
tar xvf lte_linux_x64.1.1.2.tar.bz2
cp lte_linux_x64/lte docker_dist

# build and copy worker
cd /tmp/workspace/src/worker
go build
cd /tmp
cp workspace/src/worker/worker docker_dist

# copy dependencies
cp /lib64/libdl.so.2 docker_dist
cp /lib64/librt.so.1 docker_dist

# generate dummy directories
mkdir docker_dist/shader
echo "dummuy" > docker_dist/shader/dummy
mkdir docker_dist/scene
echo "dummuy" > docker_dist/scene/dummy
