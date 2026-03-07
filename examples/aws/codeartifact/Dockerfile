FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /codeartifact ./examples/aws/codeartifact
ENTRYPOINT ["/codeartifact"]
