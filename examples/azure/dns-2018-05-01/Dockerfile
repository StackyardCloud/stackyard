FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /dns-2018-05-01-example ./examples/azure/dns-2018-05-01

ENTRYPOINT ["/dns-2018-05-01-example"]
