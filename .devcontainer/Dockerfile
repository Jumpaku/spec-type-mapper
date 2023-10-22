FROM golang:1.21.0-bookworm

ENV DEBIAN_FRONTEND=noninteractive

RUN apt update && apt install -y git curl jq


RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0 && \
    go install github.com/bufbuild/buf/cmd/buf@v1.27.1 && \
    go install github.com/google/gnostic/cmd/protoc-gen-openapi@v0.7.0

WORKDIR /work

# go tools
RUN go install golang.org/x/tools/cmd/goimports@latest
