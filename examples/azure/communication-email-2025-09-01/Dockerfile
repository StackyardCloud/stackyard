FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /communication-email-2025-09-01 ./examples/azure/communication-email-2025-09-01

ENTRYPOINT ["/communication-email-2025-09-01"]
