FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /comprehend-advanced ./examples/comprehend/comprehend-advanced
ENTRYPOINT ["/comprehend-advanced"]
