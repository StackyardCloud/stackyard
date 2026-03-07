FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /dataproc-v2-apiv1 ./examples/gcp/dataproc-v2-apiv1

ENTRYPOINT ["/dataproc-v2-apiv1"]
