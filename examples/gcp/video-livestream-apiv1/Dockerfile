FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /video-livestream-apiv1 ./examples/gcp/video-livestream-apiv1

ENTRYPOINT ["/video-livestream-apiv1"]
