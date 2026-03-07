FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /data-exchange ./examples/aws/data-exchange
CMD ["/data-exchange"]

