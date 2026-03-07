FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /maps-fleetengine-delivery-apiv1 ./examples/gcp/maps-fleetengine-delivery-apiv1

ENTRYPOINT ["/maps-fleetengine-delivery-apiv1"]
