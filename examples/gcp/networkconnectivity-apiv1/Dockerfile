FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /networkconnectivity-apiv1 ./examples/gcp/networkconnectivity-apiv1

ENTRYPOINT ["/networkconnectivity-apiv1"]
