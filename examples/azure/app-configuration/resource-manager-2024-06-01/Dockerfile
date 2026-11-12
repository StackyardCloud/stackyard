FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /resource-manager-2024-06-01 ./examples/azure/app-configuration/resource-manager-2024-06-01

ENTRYPOINT ["/resource-manager-2024-06-01"]
