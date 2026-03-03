FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /data-exchange ./examples/aws/data-exchange/data-exchange
CMD ["/data-exchange"]

