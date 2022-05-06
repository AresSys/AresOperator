# Build the manager binary
FROM golang:1.15-alpine as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY ./ ./

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o ares cmd/server.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
# FROM gcr.io/distroless/static:nonroot
FROM centos:7.3.1611
ARG HOME=/home/work
WORKDIR ${HOME}
RUN mkdir ${HOME}/conf
COPY --from=builder /workspace/ares ${HOME}
COPY --from=builder /workspace/build/*.yaml ${HOME}/conf/
CMD ["./ares"]
