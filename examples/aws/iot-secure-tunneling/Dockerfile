FROM golang:1.25-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/iot-secure-tunneling ./examples/aws/iot-secure-tunneling

CMD ["/usr/local/bin/iot-secure-tunneling"]
