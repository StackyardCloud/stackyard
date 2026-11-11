FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /cloudngfw-2025-10-08 ./examples/azure/cloudngfw-2025-10-08

ENTRYPOINT ["/cloudngfw-2025-10-08"]
