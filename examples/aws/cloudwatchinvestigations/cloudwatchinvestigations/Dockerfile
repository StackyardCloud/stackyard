FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatchinvestigations ./examples/aws/cloudwatchinvestigations/cloudwatchinvestigations
ENTRYPOINT ["/cloudwatchinvestigations"]
