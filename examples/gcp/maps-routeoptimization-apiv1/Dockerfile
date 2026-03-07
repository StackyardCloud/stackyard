FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /maps-routeoptimization-apiv1 ./examples/gcp/maps-routeoptimization-apiv1

ENTRYPOINT ["/maps-routeoptimization-apiv1"]
