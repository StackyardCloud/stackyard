FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /visionai-apiv1 ./examples/gcp/visionai-apiv1

ENTRYPOINT ["/visionai-apiv1"]
