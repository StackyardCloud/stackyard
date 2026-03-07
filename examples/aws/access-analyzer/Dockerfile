FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /access-analyzer ./examples/aws/access-analyzer
CMD ["/access-analyzer"]
