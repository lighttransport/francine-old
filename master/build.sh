#!/bin/sh

sudo docker build -t lighttransport/lte_master .

sudo docker tag lighttransport/lte_master localhost:5000/lte_master
sudo docker push localhost:5000/lte_master
