FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /compute-metadata-apiv1 ./examples/gcp/compute-metadata-apiv1

ENTRYPOINT ["/compute-metadata-apiv1"]
