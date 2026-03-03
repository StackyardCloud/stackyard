FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /fis-basic ./examples/aws/fis/fis-basic
ENTRYPOINT ["/fis-basic"]
