FROM golang:1.25-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/cloudfront-keyvaluestore ./examples/aws/cloudfront-keyvaluestore

CMD ["/usr/local/bin/cloudfront-keyvaluestore"]
