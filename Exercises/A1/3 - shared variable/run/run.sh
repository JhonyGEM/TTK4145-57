#!/bin/bash

echo "Building image from 3.Dockerfile..."
docker build -f ./3.Dockerfile -t run3img .
echo "--------------------------------------------"

echo "Running files inside container..."
docker run --rm -v "$(dirname "$(pwd)")":/app -w /app run3img bash -c "
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