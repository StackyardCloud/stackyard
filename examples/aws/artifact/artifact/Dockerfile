FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /artifact ./examples/aws/artifact/artifact
ENTRYPOINT ["/artifact"]
