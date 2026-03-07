FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /dataproc-apiv1 ./examples/gcp/dataproc-apiv1

ENTRYPOINT ["/dataproc-apiv1"]
