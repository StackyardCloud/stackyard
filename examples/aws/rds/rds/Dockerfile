FROM golang:1.23-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/rds ./examples/aws/rds/rds

CMD ["/usr/local/bin/rds"]
