FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /access-analyzer ./examples/aws/access-analyzer/access-analyzer
CMD ["/access-analyzer"]
