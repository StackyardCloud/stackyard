FROM golang:1.25-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/snow-device-management ./examples/aws/snow-device-management

CMD ["/usr/local/bin/snow-device-management"]
