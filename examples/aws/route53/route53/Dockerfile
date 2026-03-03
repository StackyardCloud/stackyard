FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /route53 ./examples/aws/route53/route53
ENTRYPOINT ["/route53"]
