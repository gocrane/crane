FROM alpine:3.13.5
RUN apk add --no-cache tzdata
WORKDIR /
COPY craned .
