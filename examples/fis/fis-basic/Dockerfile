FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /fis-basic ./examples/fis/fis-basic
ENTRYPOINT ["/fis-basic"]
