FROM golang:1.25-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/chime-sdk-media-pipelines ./examples/aws/chime-sdk-media-pipelines

CMD ["/usr/local/bin/chime-sdk-media-pipelines"]
