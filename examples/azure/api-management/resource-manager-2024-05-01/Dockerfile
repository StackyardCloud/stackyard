FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /resource-manager-2024-05-01 ./examples/azure/api-management/resource-manager-2024-05-01

ENTRYPOINT ["/resource-manager-2024-05-01"]
