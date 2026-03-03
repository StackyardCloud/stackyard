FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatchsynthetics-advanced ./examples/aws/cloudwatchsynthetics/cloudwatchsynthetics-advanced
ENTRYPOINT ["/cloudwatchsynthetics-advanced"]
