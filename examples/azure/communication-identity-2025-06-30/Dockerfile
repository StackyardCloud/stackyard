FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /communication-identity-2025-06-30 ./examples/azure/communication-identity-2025-06-30

ENTRYPOINT ["/communication-identity-2025-06-30"]
