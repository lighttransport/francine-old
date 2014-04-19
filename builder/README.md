# Light Transport Engine Docker Container Builder
## Howto

    LTE_DIR="/path/to/lte" # lte_linux_x64.1.1.2.tar.bz2 inside
    ./build_builder.sh # builds lighttransport/lte_builder
    ./run_builder.sh # builds lighttransport/lte_bin

make sure not to use sudo because it will blow LTE\_DIR env!
