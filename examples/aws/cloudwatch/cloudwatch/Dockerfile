FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatch ./examples/aws/cloudwatch/cloudwatch
ENTRYPOINT ["/cloudwatch"]
