FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /servicedirectory-apiv1 ./examples/gcp/servicedirectory-apiv1

ENTRYPOINT ["/servicedirectory-apiv1"]
