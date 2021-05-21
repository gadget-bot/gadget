#!/bin/bash

start() {
  if [[ -z ${DB_PASS} ]]; then
    echo "Missing DB_PASS env variable"
    exit 1
  fi

  if [[ -z ${DB_ROOT_PASS} ]]; then
    echo "Missing DB_ROOT_PASS env variable"
    exit 1
  fi

  docker run \
    --name gadget-mariadb \
    -v ${HOME}/.gadget/db:/var/lib/mysql \
    -e MARIADB_ROOT_PASSWORD="${DB_ROOT_PASS}" \
    -e MARIADB_DATABASE=gadget_dev \
    -e MARIADB_USER=gadget \
    -e MARIADB_PASSWORD="${DB_PASS}" \
    -p 3306:3306 \
    -d mariadb:10.5
}

stop() {
  docker stop gadget-mariadb
  docker rm gadget-mariadb
}

option=$1

case $option in
  start)
    echo "Starting DB"
    start
    ;;
  stop)
    echo "Stopping DB"
    stop
    ;;
  *)
    echo "$0 [start|stop]"
esac
