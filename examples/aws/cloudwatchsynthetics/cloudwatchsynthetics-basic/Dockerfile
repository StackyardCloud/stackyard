FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatchsynthetics-basic ./examples/aws/cloudwatchsynthetics/cloudwatchsynthetics-basic
ENTRYPOINT ["/cloudwatchsynthetics-basic"]
