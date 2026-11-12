FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /gkehub-apiv1 ./examples/gcp/gkehub-apiv1

ENTRYPOINT ["/gkehub-apiv1"]
