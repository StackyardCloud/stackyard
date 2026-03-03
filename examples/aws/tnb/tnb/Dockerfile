FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /tnb ./examples/aws/tnb/tnb
ENTRYPOINT ["/tnb"]
