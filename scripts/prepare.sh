#!/bin/bash

cd /tmp && \
   curl -LO https://go.dev/dl/go1.20.12.linux-amd64.tar.gz && \
   tar -zxf go1.20.12.linux-amd64.tar.gz && \
   sudo mv go /opt/go1.20

alias go=/opt/go1.20/bin/go

DEV_PATH=/tmp/go_path/1.20/edgevpn_dev
export GOPATH=${DEV_PATH}
mkdir -p ${DEV_PATH}

cd /home/ubuntu/edgevpn && \
  go mod tidy && \
  go build -o main.o main.go

