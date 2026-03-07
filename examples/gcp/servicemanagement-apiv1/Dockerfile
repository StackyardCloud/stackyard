FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /servicemanagement-apiv1 ./examples/gcp/servicemanagement-apiv1

ENTRYPOINT ["/servicemanagement-apiv1"]
