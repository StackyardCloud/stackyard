FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /config-apiv1 ./examples/gcp/config-apiv1

ENTRYPOINT ["/config-apiv1"]
