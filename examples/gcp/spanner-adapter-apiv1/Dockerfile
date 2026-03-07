FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /spanner-adapter-apiv1 ./examples/gcp/spanner-adapter-apiv1

ENTRYPOINT ["/spanner-adapter-apiv1"]
