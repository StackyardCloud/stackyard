FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /maps-fleetengine-apiv1 ./examples/gcp/maps-fleetengine-apiv1

ENTRYPOINT ["/maps-fleetengine-apiv1"]
