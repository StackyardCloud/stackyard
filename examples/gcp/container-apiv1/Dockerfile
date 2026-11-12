FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /container-apiv1 ./examples/gcp/container-apiv1

ENTRYPOINT ["/container-apiv1"]
