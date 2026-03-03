FROM golang:1.23-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/s3 ./examples/aws/s3/s3

CMD ["/usr/local/bin/s3"]
