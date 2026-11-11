FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatchapplicationinsights ./examples/aws/cloudwatchapplicationinsights
ENTRYPOINT ["/cloudwatchapplicationinsights"]
