FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /cloudquotas-apiv1 ./examples/gcp/cloudquotas-apiv1

ENTRYPOINT ["/cloudquotas-apiv1"]
