FROM --platform=${BUILDPLATFORM} golang:1.20 AS build

WORKDIR /app

RUN mkdir app vendor

COPY go.mod ./
COPY go.sum ./
COPY app ./app
COPY vendor ./vendor

RUN export GOOS=$(echo ${TARGETPLATFORM} | cut -d / -f1) && \
    export GOARCH=$(echo ${TARGETPLATFORM} | cut -d / -f2) && \
    go build -C app -o /ombibot

FROM --platform=${TARGETPLATFORM} gcr.io/distroless/base-debian10

WORKDIR /

COPY --from=build /ombibot /ombibot

ENTRYPOINT ["/ombibot"]
