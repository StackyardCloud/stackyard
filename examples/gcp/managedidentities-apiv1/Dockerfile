FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /managedidentities-apiv1 ./examples/gcp/managedidentities-apiv1

ENTRYPOINT ["/managedidentities-apiv1"]
