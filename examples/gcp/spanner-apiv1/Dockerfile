FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /spanner-apiv1 ./examples/gcp/spanner-apiv1

ENTRYPOINT ["/spanner-apiv1"]
