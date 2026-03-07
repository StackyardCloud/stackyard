FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /documentai-apiv1 ./examples/gcp/documentai-apiv1

ENTRYPOINT ["/documentai-apiv1"]
