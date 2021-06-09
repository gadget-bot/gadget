#!/bin/sh

export GOBIN=$HOME/go/bin
export PATH=$PATH:$GOBIN

bad() {
  echo "**************"
  echo "* $1"
  echo "**************"
  exit 1
}

go get -u golang.org/x/lint/golint

golint
lint_status=$?

if [ "$lint_status" != "0" ]; then
  bad "Linting failed"
fi

go test
test_status=$?

if [ "$test_status" != "0" ]; then
  bad "Testing failed"
fi
