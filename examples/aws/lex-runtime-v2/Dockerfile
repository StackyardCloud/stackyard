FROM golang:1.25-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/lex-runtime-v2 ./examples/aws/lex-runtime-v2

CMD ["/usr/local/bin/lex-runtime-v2"]
