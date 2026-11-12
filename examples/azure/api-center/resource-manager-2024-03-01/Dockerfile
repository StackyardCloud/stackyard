FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /resource-manager-2024-03-01 ./examples/azure/api-center/resource-manager-2024-03-01

ENTRYPOINT ["/resource-manager-2024-03-01"]
