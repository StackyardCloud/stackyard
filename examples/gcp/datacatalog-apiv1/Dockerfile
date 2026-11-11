FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /datacatalog-apiv1 ./examples/gcp/datacatalog-apiv1

ENTRYPOINT ["/datacatalog-apiv1"]
