FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /fis-advanced ./examples/aws/fis/fis-advanced
ENTRYPOINT ["/fis-advanced"]
