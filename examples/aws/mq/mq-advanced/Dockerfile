FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /mq-advanced ./examples/aws/mq/mq-advanced
ENTRYPOINT ["/mq-advanced"]
