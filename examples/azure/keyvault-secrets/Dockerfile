FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /azure-keyvault-secrets ./examples/azure/keyvault-secrets

ENTRYPOINT ["/azure-keyvault-secrets"]
