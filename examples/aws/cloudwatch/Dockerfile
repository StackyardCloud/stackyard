FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatch ./examples/aws/cloudwatch
ENTRYPOINT ["/cloudwatch"]
