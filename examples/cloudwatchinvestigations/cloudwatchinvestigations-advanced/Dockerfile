FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatchinvestigations-advanced ./examples/cloudwatchinvestigations/cloudwatchinvestigations-advanced
ENTRYPOINT ["/cloudwatchinvestigations-advanced"]
