FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /kinesisvideostreams ./examples/aws/kinesisvideostreams
ENTRYPOINT ["/kinesisvideostreams"]
