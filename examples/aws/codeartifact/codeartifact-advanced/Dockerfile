FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codeartifact-advanced ./examples/aws/codeartifact/codeartifact-advanced
ENTRYPOINT ["/codeartifact-advanced"]
