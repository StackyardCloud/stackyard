FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /account-management-2025-06-01 ./examples/azure/ai-foundry/account-management-2025-06-01

ENTRYPOINT ["/account-management-2025-06-01"]
