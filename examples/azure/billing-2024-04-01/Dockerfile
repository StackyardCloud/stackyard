FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /billing-2024-04-01 ./examples/azure/billing-2024-04-01

ENTRYPOINT ["/billing-2024-04-01"]
