FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /devcenter-administrator-2024-02-01-example ./examples/azure/devcenter-administrator-2024-02-01

ENTRYPOINT ["/devcenter-administrator-2024-02-01-example"]
