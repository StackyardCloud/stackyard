FROM golang:1.23-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/sesv2 ./examples/aws/sesv2/sesv2

CMD ["/usr/local/bin/sesv2"]
