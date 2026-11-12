FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatchsynthetics ./examples/aws/cloudwatchsynthetics
ENTRYPOINT ["/cloudwatchsynthetics"]
