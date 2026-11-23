FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /timestream-liveanalytics ./examples/aws/timestream-liveanalytics
ENTRYPOINT ["/timestream-liveanalytics"]
