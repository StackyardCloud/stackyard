FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /synapse-2021-06-01 ./examples/azure/synapse-2021-06-01

ENTRYPOINT ["/synapse-2021-06-01"]
