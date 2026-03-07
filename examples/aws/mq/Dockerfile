FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /mq ./examples/aws/mq
ENTRYPOINT ["/mq"]
