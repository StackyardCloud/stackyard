FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /route53 ./examples/aws/route53
ENTRYPOINT ["/route53"]
