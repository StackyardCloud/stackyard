FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /advisor-2023-01-01 ./examples/azure/advisor-2023-01-01

ENTRYPOINT ["/advisor-2023-01-01"]
