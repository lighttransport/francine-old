ABS_SH=`readlink -f $0`
ABS_DIR=`dirname $ABS_SH`
sudo docker run -d -v $ABS_DIR/models:/tmp/models -e MODEL_DIR=/tmp/models -w /tmp -p 8080:8080 -e LTE_HOST=107.178.213.74 lighttransport/rotate_demo /bin/rotate_demo
