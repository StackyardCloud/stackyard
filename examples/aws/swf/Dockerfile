FROM golang:1.24-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/swf ./examples/aws/swf

CMD ["/usr/local/bin/swf"]
