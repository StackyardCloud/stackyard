FROM golang:1.25-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/freetier ./examples/aws/freetier

CMD ["/usr/local/bin/freetier"]
