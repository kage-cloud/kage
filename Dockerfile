FROM golang:1.14 as builder

ARG BUILD_DIR=xds

COPY ${BUILD_DIR} .

RUN go build -o main

FROM alpine

COPY --from=builder main .

RUN main
