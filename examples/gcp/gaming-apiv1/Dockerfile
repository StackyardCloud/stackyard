FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /gaming-apiv1 ./examples/gcp/gaming-apiv1

ENTRYPOINT ["/gaming-apiv1"]
