FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /discoveryengine-apiv1 ./examples/gcp/discoveryengine-apiv1

ENTRYPOINT ["/discoveryengine-apiv1"]
