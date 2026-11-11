FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /playwright-dataplane-2025-09-01 ./examples/azure/playwright-dataplane-2025-09-01

ENTRYPOINT ["/playwright-dataplane-2025-09-01"]
