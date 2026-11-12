FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /attestation-2025-06-01 ./examples/azure/attestation-2025-06-01

ENTRYPOINT ["/attestation-2025-06-01"]
