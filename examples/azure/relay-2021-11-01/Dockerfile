FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /relay-2021-11-01 ./examples/azure/relay-2021-11-01

ENTRYPOINT ["/relay-2021-11-01"]
