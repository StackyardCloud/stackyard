FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /data-exchange-basic ./examples/aws/data-exchange/data-exchange-basic
CMD ["/data-exchange-basic"]

