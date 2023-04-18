FROM --platform=${BUILDPLATFORM} golang:1.20 AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

RUN mkdir app

COPY go.mod ./
COPY go.sum ./
COPY app ./app

RUN  CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -C app -o /ombibot

FROM --platform=${TARGETPLATFORM} gcr.io/distroless/base-debian10

WORKDIR /

COPY --from=build /ombibot /ombibot

ENTRYPOINT ["/ombibot"]
