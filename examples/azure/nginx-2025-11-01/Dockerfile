FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /nginx-2025-11-01 ./examples/azure/nginx-2025-11-01

ENTRYPOINT ["/nginx-2025-11-01"]
