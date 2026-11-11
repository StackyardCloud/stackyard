FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /containerapps-2025-10-02-preview ./examples/azure/containerapps-2025-10-02-preview

ENTRYPOINT ["/containerapps-2025-10-02-preview"]
