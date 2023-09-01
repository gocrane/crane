ARG PKGNAME

# Build the manager binary
FROM golang:1.17.2-alpine as builder

ARG LDFLAGS
ARG PKGNAME
ARG BUILD

ENV https_proxy http://192.168.64.2:7890
ENV http_proxy http://192.168.64.2:7890
ENV all_proxy socks5://192.168.64.2:7890

WORKDIR /go/src/github.com/gocrane/crane

# Add build deps
RUN apk add build-base

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN if [[ "${BUILD}" != "CI" ]]; then go env -w GOPROXY=https://goproxy.cn,direct; fi
RUN go env
RUN go mod download

# Copy the go source
COPY pkg pkg/
COPY cmd cmd/

# Build
RUN env
RUN go build -ldflags="${LDFLAGS}" -a -o ${PKGNAME} /go/src/github.com/gocrane/crane/cmd/${PKGNAME}/main.go
FROM alpine:3.13.5
ENV https_proxy http://192.168.64.2:7890
ENV http_proxy http://192.168.64.2:7890
ENV all_proxy socks5://192.168.64.2:7890

RUN apk add --no-cache tzdata
WORKDIR /
ARG PKGNAME
COPY --from=builder /go/src/github.com/gocrane/crane/${PKGNAME} .
ENV https_proxy ""
ENV http_proxy ""
ENV all_proxy ""
