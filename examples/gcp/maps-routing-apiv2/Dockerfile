FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /maps-routing-apiv2 ./examples/gcp/maps-routing-apiv2

ENTRYPOINT ["/maps-routing-apiv2"]
