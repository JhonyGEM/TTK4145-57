#!/bin/bash

echo "Building image from 4.Dockerfile ..."
docker build -f ./4.Dockerfile -t run4img .
echo "--------------------------------------------"

echo "Running files inside container ..."
docker run --rm -v "$(dirname "$(pwd)")":/app -w /app run4img bash -c "
    echo
    cd c
    gcc main.c ringbuf.c -o main
    echo
    echo 'Running main.go'
    echo '--------------------------------------------'
    echo
    cd ../go
    go run main.go
    echo '--------------------------------------------'
    echo
    echo
    echo 'Running main.c'
    echo '--------------------------------------------'
    echo
    cd ../c
    ./main
    echo '--------------------------------------------'
    echo
"