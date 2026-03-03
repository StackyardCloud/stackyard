FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /comprehend-basic ./examples/aws/comprehend/comprehend-basic
ENTRYPOINT ["/comprehend-basic"]
