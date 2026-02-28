FROM golang:1.23-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/ebs-basic ./examples/ebs/ebs-basic

CMD ["/usr/local/bin/ebs-basic"]
