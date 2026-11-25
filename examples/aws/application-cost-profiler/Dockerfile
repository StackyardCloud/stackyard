FROM golang:1.25-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/application-cost-profiler ./examples/aws/application-cost-profiler

CMD ["/usr/local/bin/application-cost-profiler"]
