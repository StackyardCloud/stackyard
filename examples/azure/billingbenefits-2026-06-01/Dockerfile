FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /billingbenefits-2026-06-01 ./examples/azure/billingbenefits-2026-06-01

ENTRYPOINT ["/billingbenefits-2026-06-01"]
