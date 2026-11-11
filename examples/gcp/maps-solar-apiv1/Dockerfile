FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /maps-solar-apiv1 ./examples/gcp/maps-solar-apiv1

ENTRYPOINT ["/maps-solar-apiv1"]
