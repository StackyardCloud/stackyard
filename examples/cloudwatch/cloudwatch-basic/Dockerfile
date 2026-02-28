FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatch-basic ./examples/cloudwatch/cloudwatch-basic
ENTRYPOINT ["/cloudwatch-basic"]
