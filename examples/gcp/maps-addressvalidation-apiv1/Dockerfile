FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /maps-addressvalidation-apiv1 ./examples/gcp/maps-addressvalidation-apiv1

ENTRYPOINT ["/maps-addressvalidation-apiv1"]
