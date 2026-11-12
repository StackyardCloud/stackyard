FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /carbon-2025-04-01 ./examples/azure/carbon-2025-04-01

ENTRYPOINT ["/carbon-2025-04-01"]
