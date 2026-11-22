FROM golang:1.25-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/route53domains ./examples/aws/route53domains

CMD ["/usr/local/bin/route53domains"]
