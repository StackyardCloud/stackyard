FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /azurerekusto-2025-02-14 ./examples/azure/azurerekusto-2025-02-14

ENTRYPOINT ["/azurerekusto-2025-02-14"]
