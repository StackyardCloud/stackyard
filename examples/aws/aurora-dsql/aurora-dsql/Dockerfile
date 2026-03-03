FROM golang:1.23-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/aurora-dsql ./examples/aws/aurora-dsql/aurora-dsql

CMD ["/usr/local/bin/aurora-dsql"]
