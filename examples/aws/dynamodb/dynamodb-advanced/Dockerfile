FROM golang:1.23-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/dynamodb-advanced ./examples/aws/dynamodb/dynamodb-advanced

CMD ["/usr/local/bin/dynamodb-advanced"]
