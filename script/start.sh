#!/bin/sh -e

# # before running this script, do this to start postgres:
# docker run -it -p 5432:5432 -e POSTGRES_HOST_AUTH_METHOD=trust -e POSTGRES_USER=feedback postgres:13
# # then this setup the database :
# docker run -it -p 5432:5432 -e POSTGRES_HOST_AUTH_METHOD=trust -e POSTGRES_USER=feedback postgres:13
# docker run -it postgres psql -h host.docker.internal -U feedback -d feedback -c 'CREATE EXTENSION pgcrypto;'

docker build . -t test/feedback:local

docker run -e POSTGRES_CONNECTION_STRING=postgres://feedback@host.docker.internal:5432/feedback?sslmode=disable \
	-p 8080:8080 test/feedback:local
