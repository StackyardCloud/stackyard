FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /account-management-2024-10-01 ./examples/azure/ai-services/account-management-2024-10-01

ENTRYPOINT ["/account-management-2024-10-01"]
