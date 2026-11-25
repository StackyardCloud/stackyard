FROM golang:1.25-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/connect-contact-lens ./examples/aws/connect-contact-lens

CMD ["/usr/local/bin/connect-contact-lens"]
