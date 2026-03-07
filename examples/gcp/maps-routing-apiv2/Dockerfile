FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /maps-routing-apiv2 ./examples/gcp/maps-routing-apiv2

ENTRYPOINT ["/maps-routing-apiv2"]
