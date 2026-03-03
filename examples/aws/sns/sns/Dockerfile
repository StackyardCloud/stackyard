FROM golang:1.23-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/sns ./examples/aws/sns/sns

CMD ["/usr/local/bin/sns"]
