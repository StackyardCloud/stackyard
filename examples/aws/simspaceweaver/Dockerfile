FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /simspaceweaver ./examples/aws/simspaceweaver
ENTRYPOINT ["/simspaceweaver"]
