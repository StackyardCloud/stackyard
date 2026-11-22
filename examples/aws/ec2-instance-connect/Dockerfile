FROM golang:1.25-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/ec2-instance-connect ./examples/aws/ec2-instance-connect

CMD ["/usr/local/bin/ec2-instance-connect"]
