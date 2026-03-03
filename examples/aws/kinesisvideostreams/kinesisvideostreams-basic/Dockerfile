FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /kinesisvideostreams-basic ./examples/aws/kinesisvideostreams/kinesisvideostreams-basic
ENTRYPOINT ["/kinesisvideostreams-basic"]
