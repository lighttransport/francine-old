# Francine

![Teaser](https://github.com/lighttransport/francine/blob/master/img/sakura.jpg?raw=true)

Renderer backend for massive server environment.

## Usage

### Build master
    cd master
    ./build.sh

### Build demo
    cd demo
    ./build.sh

### Build worker
    cd worker
    LTE_VERSION="1.1.2" # set lte version
    LTE_DIR="/path/to/lte" # lte_linux_x64.$(LTE_VERSION).tar.bz2 inside
    ./build.sh

### Create master GCE instance
    cd ltesetup
    go build
    ./ltesetup create_master

### Update images
    # make sure you have built master, demo and worker images before doing this
    ./ltesetup update_images

### Create worker GCE instance
    ./ltesetup create_worker
