FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /codeartifact ./examples/aws/codeartifact
ENTRYPOINT ["/codeartifact"]
