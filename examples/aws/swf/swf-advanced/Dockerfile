FROM golang:1.23-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/swf-advanced ./examples/aws/swf/swf-advanced

CMD ["/usr/local/bin/swf-advanced"]
