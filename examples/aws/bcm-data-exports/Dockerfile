FROM golang:1.25-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/bcm-data-exports ./examples/aws/bcm-data-exports

CMD ["/usr/local/bin/bcm-data-exports"]
