FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /comprehend ./examples/aws/comprehend
ENTRYPOINT ["/comprehend"]
