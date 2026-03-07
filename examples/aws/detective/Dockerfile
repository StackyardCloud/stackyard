FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /detective ./examples/aws/detective
ENTRYPOINT ["/detective"]
