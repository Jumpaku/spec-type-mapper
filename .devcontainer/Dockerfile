FROM golang:1.21.0-bookworm

ENV DEBIAN_FRONTEND=noninteractive

RUN apt update && apt install -y git curl jq

WORKDIR /work

# go tools
RUN go install golang.org/x/tools/cmd/goimports@latest
