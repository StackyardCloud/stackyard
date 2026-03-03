FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /simspaceweaver ./examples/aws/simspaceweaver/simspaceweaver
ENTRYPOINT ["/simspaceweaver"]
