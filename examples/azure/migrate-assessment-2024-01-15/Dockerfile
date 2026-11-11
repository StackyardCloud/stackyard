FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /migrate-assessment-2024-01-15 ./examples/azure/migrate-assessment-2024-01-15

ENTRYPOINT ["/migrate-assessment-2024-01-15"]
