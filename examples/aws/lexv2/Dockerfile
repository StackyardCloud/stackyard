FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /lexv2 ./examples/aws/lexv2
ENTRYPOINT ["/lexv2"]
