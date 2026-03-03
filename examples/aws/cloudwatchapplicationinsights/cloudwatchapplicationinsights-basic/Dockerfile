FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatchapplicationinsights-basic ./examples/aws/cloudwatchapplicationinsights/cloudwatchapplicationinsights-basic
ENTRYPOINT ["/cloudwatchapplicationinsights-basic"]
