FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /resource-manager-2025-05-01 ./examples/azure/app-service/resource-manager-2025-05-01

ENTRYPOINT ["/resource-manager-2025-05-01"]
