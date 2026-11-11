FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudfront ./examples/aws/cloudfront
ENTRYPOINT ["/cloudfront"]
