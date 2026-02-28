FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatchapplicationinsights-advanced ./examples/cloudwatchapplicationinsights/cloudwatchapplicationinsights-advanced
ENTRYPOINT ["/cloudwatchapplicationinsights-advanced"]
