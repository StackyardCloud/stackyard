FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /maps-fleetengine-delivery-apiv1 ./examples/gcp/maps-fleetengine-delivery-apiv1

ENTRYPOINT ["/maps-fleetengine-delivery-apiv1"]
