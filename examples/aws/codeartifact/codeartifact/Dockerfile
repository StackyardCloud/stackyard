FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codeartifact ./examples/aws/codeartifact/codeartifact
ENTRYPOINT ["/codeartifact"]
