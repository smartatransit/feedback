#!/bin/sh -e

docker build . -t test/feedback:local

# docker run -it -p 5432:5432 -e POSTGRES_HOST_AUTH_METHOD=trust -e POSTGRES_USER=feedback postgres:13

docker run --env-file .env -p 8080:8080 test/feedback:local
