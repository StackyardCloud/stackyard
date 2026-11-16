FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /quota-2025-09-01 ./examples/azure/quota-2025-09-01

ENTRYPOINT ["/quota-2025-09-01"]
