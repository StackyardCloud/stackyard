FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ram-advanced ./examples/aws/ram/ram-advanced
ENTRYPOINT ["/ram-advanced"]
