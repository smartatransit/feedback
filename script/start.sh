#!/bin/sh

docker build . -t test/feedback:local

docker run --env-file .env -p 8080 test/feedback:local
