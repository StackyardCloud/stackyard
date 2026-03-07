FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /transcribe ./examples/aws/transcribe
ENTRYPOINT ["/transcribe"]
