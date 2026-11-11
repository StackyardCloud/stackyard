FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /dns-privatedns-2018-09-01-example ./examples/azure/dns-privatedns-2018-09-01

ENTRYPOINT ["/dns-privatedns-2018-09-01-example"]
