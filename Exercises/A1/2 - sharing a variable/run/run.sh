#!/bin/bash

echo "Building image from 2.Dockerfile..."
docker build -f ./2.Dockerfile -t run2img .
echo "--------------------------------------------"

echo "Running files inside container..."
docker run --rm -v "$(dirname "$(pwd)")":/app -w /app run2img bash -c "
    echo
    cd c
    gcc foo.c -o foo
    echo
    echo 'Running foo.go ...'
    echo
    cd ../go
    go run foo.go
    echo '--------------------------------------------'
    echo
    echo
    echo 'Running foo.c ...'
    echo
    cd ../c
    ./foo
    echo '--------------------------------------------'
    echo
"