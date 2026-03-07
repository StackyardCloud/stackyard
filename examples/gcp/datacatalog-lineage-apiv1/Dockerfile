FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /datacatalog-lineage-apiv1 ./examples/gcp/datacatalog-lineage-apiv1

ENTRYPOINT ["/datacatalog-lineage-apiv1"]
