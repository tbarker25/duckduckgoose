# syntax=docker/dockerfile:1

FROM golang:1.20-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./
COPY gen ./gen

RUN go build -o /duckduckgoose

EXPOSE 8080 9090

CMD [ "/duckduckgoose", "-raft_addr=0.0.0.0:9090", "-api_addr=0.0.0.0:8080", "-state_dir=/tmp/duckduckgoose", "-bootstrap_cluster=true"]

