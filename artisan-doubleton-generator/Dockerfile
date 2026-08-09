FROM golang:1.25.0 AS build
LABEL authors="cameronroe"

WORKDIR /project

COPY . .
RUN go mod download
RUN go build -o app ./...

ENTRYPOINT ["/project/app"]
