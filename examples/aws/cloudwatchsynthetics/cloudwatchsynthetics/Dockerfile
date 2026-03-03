FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatchsynthetics ./examples/aws/cloudwatchsynthetics/cloudwatchsynthetics
ENTRYPOINT ["/cloudwatchsynthetics"]
