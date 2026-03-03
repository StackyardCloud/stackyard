FROM golang:1.23-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/rds-advanced ./examples/aws/rds/rds-advanced

CMD ["/usr/local/bin/rds-advanced"]
