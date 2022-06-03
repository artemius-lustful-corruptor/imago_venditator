# syntax=docker/dockerfile:1

FROM golang:1.16-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./
COPY config.json ./
COPY entrypoint.sh ./
RUN go build -o image-vendicator

CMD ["./entrypoint.sh"]
