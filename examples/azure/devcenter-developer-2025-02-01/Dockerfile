FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /devcenter-developer-2025-02-01-example ./examples/azure/devcenter-developer-2025-02-01

ENTRYPOINT ["/devcenter-developer-2025-02-01-example"]
