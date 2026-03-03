FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /access-analyzer-advanced ./examples/aws/access-analyzer/access-analyzer-advanced
CMD ["/access-analyzer-advanced"]
