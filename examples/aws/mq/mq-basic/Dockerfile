FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /mq-basic ./examples/aws/mq/mq-basic
ENTRYPOINT ["/mq-basic"]
