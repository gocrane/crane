ARG PKGNAME

# Build the manager binary
FROM golang:1.17.2-alpine as builder

ARG LDFLAGS
ARG PKGNAME

WORKDIR /go/src/github.com/gocrane/crane
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go env -w GOPROXY=https://goproxy.io,direct
RUN go mod download

# Copy the go source
COPY pkg pkg/
COPY cmd cmd/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS}" -a -o ${PKGNAME} /go/src/github.com/gocrane/crane/cmd/${PKGNAME}/main.go
FROM alpine:3.13.5
RUN apk add --no-cache tzdata
WORKDIR /
ARG PKGNAME
COPY --from=builder /go/src/github.com/gocrane/crane/${PKGNAME} .