FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /maps-areainsights-apiv1 ./examples/gcp/maps-areainsights-apiv1

ENTRYPOINT ["/maps-areainsights-apiv1"]
