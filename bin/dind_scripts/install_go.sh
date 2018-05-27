#!/bin/bash
if ! [ -x "$(command -v go)" ]; then
    wget https://dl.google.com/go/go1.10.1.linux-amd64.tar.gz
    sudo tar -xvf go1.10.1.linux-amd64.tar.gz
    sudo mv go /usr/local
    mkdir -p ~/go/src
    mkdir -p ~/go/bin
    export GOROOT=/usr/local/go
    echo "export GOROOT=/usr/local/go" >> ~/.profile
    export GOPATH=$HOME/go
    echo "export GOPATH=$HOME/go" >> ~/.profile
    export PATH=\$GOPATH/bin:\$GOROOT/bin:\$PATH
    echo "export PATH=\$GOPATH/bin:\$GOROOT/bin:\$PATH" >> ~/.profile
    source ~/.profile
    go version && go env
    echo "Go installed. If you did not run this script in your current shell, i.e. \". install_go.sh\", please run \"source ~/.profile\" now."
else
   echo "Go already installed."
fi

if ! [ -x "$(command -v glide)" ]; then
    curl https://glide.sh/get | sh
else
   echo "Glide already installed."
fi