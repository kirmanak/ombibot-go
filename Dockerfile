ARG TARGETOS
ARG TARGETARCH

FROM --platform=${BUILDPLATFORM} golang:1.20 AS base

FROM base AS build-amd64
ENV GOOS=linux GOARCH=amd64

FROM base AS build-arm64
ENV CC=arm-linux-gnueabihf-gcc CXX=arm-linux-gnueabihf-g++ \
    CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=7

FROM build-${TARGETARCH} AS build

WORKDIR /app

RUN mkdir app vendor

COPY go.mod ./
COPY go.sum ./
COPY app ./app
COPY vendor ./vendor

RUN go build -C app -o /ombibot

FROM --platform=${TARGETPLATFORM} gcr.io/distroless/base-debian10

WORKDIR /

COPY --from=build /ombibot /ombibot

ENTRYPOINT ["/ombibot"]
