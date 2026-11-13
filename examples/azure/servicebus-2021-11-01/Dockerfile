FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /servicebus-2021-11-01 ./examples/azure/servicebus-2021-11-01

ENTRYPOINT ["/servicebus-2021-11-01"]
