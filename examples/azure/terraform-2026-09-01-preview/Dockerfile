FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /terraform-2026-09-01-preview ./examples/azure/terraform-2026-09-01-preview

ENTRYPOINT ["/terraform-2026-09-01-preview"]
