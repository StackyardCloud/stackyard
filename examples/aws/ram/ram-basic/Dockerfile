FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ram-basic ./examples/aws/ram/ram-basic
ENTRYPOINT ["/ram-basic"]
