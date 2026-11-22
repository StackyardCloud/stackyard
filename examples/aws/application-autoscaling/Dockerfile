FROM golang:1.25-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/application-autoscaling ./examples/aws/application-autoscaling

CMD ["/usr/local/bin/application-autoscaling"]
