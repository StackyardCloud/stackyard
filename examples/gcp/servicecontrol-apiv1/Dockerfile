FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /servicecontrol-apiv1 ./examples/gcp/servicecontrol-apiv1

ENTRYPOINT ["/servicecontrol-apiv1"]
