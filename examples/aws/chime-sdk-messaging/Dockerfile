FROM golang:1.25-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/chime-sdk-messaging ./examples/aws/chime-sdk-messaging

CMD ["/usr/local/bin/chime-sdk-messaging"]
