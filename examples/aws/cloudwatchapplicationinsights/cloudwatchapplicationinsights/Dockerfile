FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatchapplicationinsights ./examples/aws/cloudwatchapplicationinsights/cloudwatchapplicationinsights
ENTRYPOINT ["/cloudwatchapplicationinsights"]
