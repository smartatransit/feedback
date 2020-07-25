FROM golang:1.14 AS builder

WORKDIR /src
COPY go.mod go.mod
COPY go.sum go.sum
COPY main.go main.go
COPY db/ db/
COPY api/ api/
COPY vendor/ vendor/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -a -installsuffix cgo

FROM alpine:3.11
RUN apk --no-cache add ca-certificates

COPY --from=builder /src/feedback /bin/feedback
COPY db-migrations/ /db-migrations/
CMD ["/bin/feedback"]
