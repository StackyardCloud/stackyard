FROM golang:1.25-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/geo-places ./examples/aws/geo-places

CMD ["/usr/local/bin/geo-places"]
