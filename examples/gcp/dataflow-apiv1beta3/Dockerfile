FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /dataflow-apiv1beta3 ./examples/gcp/dataflow-apiv1beta3

ENTRYPOINT ["/dataflow-apiv1beta3"]
