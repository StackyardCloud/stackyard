FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /video-transcoder-apiv1 ./examples/gcp/video-transcoder-apiv1

ENTRYPOINT ["/video-transcoder-apiv1"]
