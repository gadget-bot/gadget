#!/bin/sh

export GOBIN=$HOME/go/bin
export PATH=$PATH:$GOBIN

make tools lint test
