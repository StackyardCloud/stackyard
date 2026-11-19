FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /newrelic-2026-06-01 ./examples/azure/newrelic-2026-06-01

ENTRYPOINT ["/newrelic-2026-06-01"]
