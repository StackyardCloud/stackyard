FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /edgenetwork-apiv1 ./examples/gcp/edgenetwork-apiv1

ENTRYPOINT ["/edgenetwork-apiv1"]
