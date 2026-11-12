FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /parametermanager-apiv1 ./examples/gcp/parametermanager-apiv1

ENTRYPOINT ["/parametermanager-apiv1"]
