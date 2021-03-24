#!/bin/bash

cd `dirname $0`
GOSAMPLE=${1:-samples/cpu/main.go}

if [ `go get ... 2>&1 | wc -l` -ne 0 ]; then
  go mod tidy
  docker-compose down
fi
docker-compose ps | egrep -q "goapp.* Up "
if [ $? -ne 0 ]; then
  docker-compose up -d --build
fi

docker-compose exec goapp go run $GOSAMPLE
# docker-compose exec goapp bash
