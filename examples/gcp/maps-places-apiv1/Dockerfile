FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /maps-places-apiv1 ./examples/gcp/maps-places-apiv1

ENTRYPOINT ["/maps-places-apiv1"]
