FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /aks-2025-10-01 ./examples/azure/aks-2025-10-01

ENTRYPOINT ["/aks-2025-10-01"]
