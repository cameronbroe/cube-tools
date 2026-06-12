FROM golang:1.26.4 AS build
LABEL authors="cameronroe"

WORKDIR /project

COPY . .
RUN go mod download
RUN go build -o app ./...

ENTRYPOINT ["/project/app"]
