#!/bin/sh

# Don't forget to run
# $ gcloud components update preview
# to enable replica pools feature

gcloud preview replica-pools --zone us-central1-a  create --size 100 --template my-replica-pool-template.json my-pool
