FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /dashboard-2025-08-01 ./examples/azure/dashboard-2025-08-01

ENTRYPOINT ["/dashboard-2025-08-01"]
