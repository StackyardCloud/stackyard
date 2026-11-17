FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /peering-2025-05-01 ./examples/azure/peering-2025-05-01

ENTRYPOINT ["/peering-2025-05-01"]
