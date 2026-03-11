FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /app-compliance-2024-06-27 ./examples/azure/app-compliance-2024-06-27

ENTRYPOINT ["/app-compliance-2024-06-27"]
