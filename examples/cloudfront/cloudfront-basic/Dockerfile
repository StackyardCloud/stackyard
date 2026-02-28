FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudfront-basic ./examples/cloudfront/cloudfront-basic
ENTRYPOINT ["/cloudfront-basic"]
