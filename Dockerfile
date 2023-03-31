FROM golang:1.20 AS build

WORKDIR /app

RUN mkdir app vendor

COPY go.mod ./
COPY go.sum ./
COPY app ./app
COPY vendor ./vendor

RUN go build -C app -o /ombibot

FROM gcr.io/distroless/base-debian10

WORKDIR /

COPY --from=build /ombibot /ombibot

ENTRYPOINT ["/ombibot"]