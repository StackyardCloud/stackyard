FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /comprehend-advanced ./examples/aws/comprehend/comprehend-advanced
ENTRYPOINT ["/comprehend-advanced"]
