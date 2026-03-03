FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /dlm ./examples/aws/dlm/dlm
CMD ["/dlm"]
