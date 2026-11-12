FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /discovery-2026-06-01-example ./examples/azure/discovery-2026-06-01

ENTRYPOINT ["/discovery-2026-06-01-example"]
