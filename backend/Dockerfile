FROM golang:1.25-alpine AS build

WORKDIR /app
COPY go.mod ./
COPY go.sum ./
COPY cmd ./cmd
COPY internal ./internal

RUN go build -o /out/invest-api ./cmd/api

FROM alpine:3.22

WORKDIR /app
COPY --from=build /out/invest-api /usr/local/bin/invest-api

EXPOSE 8080

ENTRYPOINT ["invest-api"]
