FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /playwright-resourcemanager-2025-09-01 ./examples/azure/playwright-resourcemanager-2025-09-01

ENTRYPOINT ["/playwright-resourcemanager-2025-09-01"]
