FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /fsx-advanced ./examples/aws/fsx/fsx-advanced
ENTRYPOINT ["/fsx-advanced"]
