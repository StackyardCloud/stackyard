FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /gkehub-apiv1 ./examples/gcp/gkehub-apiv1

ENTRYPOINT ["/gkehub-apiv1"]
