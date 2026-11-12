FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /data-plane-2024-09-01 ./examples/azure/app-configuration/data-plane-2024-09-01

ENTRYPOINT ["/data-plane-2024-09-01"]
