FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /datastream-apiv1 ./examples/gcp/datastream-apiv1

ENTRYPOINT ["/datastream-apiv1"]
