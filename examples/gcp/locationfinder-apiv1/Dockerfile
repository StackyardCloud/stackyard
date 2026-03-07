FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /locationfinder-apiv1 ./examples/gcp/locationfinder-apiv1

ENTRYPOINT ["/locationfinder-apiv1"]
