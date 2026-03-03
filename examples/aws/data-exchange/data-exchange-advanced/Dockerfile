FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /data-exchange-advanced ./examples/aws/data-exchange/data-exchange-advanced
CMD ["/data-exchange-advanced"]

