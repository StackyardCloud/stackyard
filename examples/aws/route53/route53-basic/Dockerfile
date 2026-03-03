FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /route53-basic ./examples/aws/route53/route53-basic
ENTRYPOINT ["/route53-basic"]
