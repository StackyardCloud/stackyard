FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /secretmanager-apiv1 ./examples/gcp/secretmanager-apiv1

ENTRYPOINT ["/secretmanager-apiv1"]
