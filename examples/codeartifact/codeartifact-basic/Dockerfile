FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codeartifact-basic ./examples/codeartifact/codeartifact-basic
ENTRYPOINT ["/codeartifact-basic"]
