FROM golang:1.23-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/s3-basic ./examples/s3/s3-basic

CMD ["/usr/local/bin/s3-basic"]
