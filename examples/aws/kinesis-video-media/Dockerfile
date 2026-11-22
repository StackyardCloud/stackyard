FROM golang:1.25-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/kinesis-video-media ./examples/aws/kinesis-video-media

CMD ["/usr/local/bin/kinesis-video-media"]
