FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /detective-advanced ./examples/aws/detective/detective-advanced
ENTRYPOINT ["/detective-advanced"]
