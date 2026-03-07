FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /artifact ./examples/aws/artifact
ENTRYPOINT ["/artifact"]
