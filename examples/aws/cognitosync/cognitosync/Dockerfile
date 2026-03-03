FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cognitosync ./examples/aws/cognitosync/cognitosync
ENTRYPOINT ["/cognitosync"]
