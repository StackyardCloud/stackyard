FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /gkeconnect-apiv1 ./examples/gcp/gkeconnect-apiv1

ENTRYPOINT ["/gkeconnect-apiv1"]
