FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /dlm ./examples/aws/dlm
CMD ["/dlm"]
