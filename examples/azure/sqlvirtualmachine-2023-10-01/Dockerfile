FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /sqlvirtualmachine-2023-10-01 ./examples/azure/sqlvirtualmachine-2023-10-01

ENTRYPOINT ["/sqlvirtualmachine-2023-10-01"]
