#!/bin/bash
if ! [ -x "$(command -v docker)" ]; then
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
    sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
    sudo apt-get update && sudo apt-get install -y docker-ce
    sudo systemctl status docker
    sudo usermod -aG docker ${USER}
else
   echo "Docker already installed."
fi

if ! [ -x "$(command -v docker-compose)" ]; then
    sudo curl -L https://github.com/docker/compose/releases/download/1.18.0/docker-compose-`uname -s`-`uname -m` -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
else
   echo "Docker Compose already installed."
fi