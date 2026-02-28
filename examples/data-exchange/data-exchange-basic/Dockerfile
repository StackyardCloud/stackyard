FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /data-exchange-basic ./examples/data-exchange/data-exchange-basic
CMD ["/data-exchange-basic"]

