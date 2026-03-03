FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudfront-advanced ./examples/aws/cloudfront/cloudfront-advanced
ENTRYPOINT ["/cloudfront-advanced"]
