# -*- mode: ruby -*-
# vi: set ft=ruby :

VAGRANTFILE_API_VERSION = "2"

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
  config.vm.box = "ubuntu/trusty64"

  config.vm.provision "shell", privileged: false, inline: <<SCRIPT
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys 36A1D7869245C8950F966E92D8576A8BA88D21E9
sudo sh -c "echo deb https://get.docker.io/ubuntu docker main > /etc/apt/sources.list.d/docker.list"
sudo apt-get update -y
sudo apt-get install -y lxc-docker
sudo apt-get install -y mercurial wget git
cd /home/vagrant && wget --quiet --no-check-certificate https://storage.googleapis.com/golang/go1.2.2.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf /home/vagrant/go1.2.2.linux-amd64.tar.gz
mkdir /home/vagrant/workspace
echo "export GOPATH=/home/vagrant/workspace" >> .bashrc
echo "export PATH=\$PATH:/usr/local/go/bin" >> .bashrc
PATH=$PATH:/usr/local/go/bin
GOPATH=/home/vagrant/workspace go get github.com/garyburd/redigo/redis
GOPATH=/home/vagrant/workspace go get code.google.com/p/goauth2/oauth
SCRIPT

  config.vm.network "forwarded_port", guest: 7000, host: 7000, protocol: 'tcp'
end
