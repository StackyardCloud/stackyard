FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatchinvestigations-basic ./examples/cloudwatchinvestigations/cloudwatchinvestigations-basic
ENTRYPOINT ["/cloudwatchinvestigations-basic"]
