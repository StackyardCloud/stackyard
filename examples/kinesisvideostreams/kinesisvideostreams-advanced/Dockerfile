FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /kinesisvideostreams-advanced ./examples/kinesisvideostreams/kinesisvideostreams-advanced
ENTRYPOINT ["/kinesisvideostreams-advanced"]
