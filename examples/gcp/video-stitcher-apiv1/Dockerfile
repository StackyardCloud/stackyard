FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /video-stitcher-apiv1 ./examples/gcp/video-stitcher-apiv1

ENTRYPOINT ["/video-stitcher-apiv1"]
