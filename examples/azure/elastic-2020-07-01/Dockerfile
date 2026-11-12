FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /elastic-2020-07-01-example ./examples/azure/elastic-2020-07-01

ENTRYPOINT ["/elastic-2020-07-01-example"]
