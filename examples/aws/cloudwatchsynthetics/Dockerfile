FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatchsynthetics ./examples/aws/cloudwatchsynthetics
ENTRYPOINT ["/cloudwatchsynthetics"]
