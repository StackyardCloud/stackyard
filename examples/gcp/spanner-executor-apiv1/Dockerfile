FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /spanner-executor-apiv1 ./examples/gcp/spanner-executor-apiv1

ENTRYPOINT ["/spanner-executor-apiv1"]
