FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /lexv2 ./examples/aws/lexv2/lexv2
ENTRYPOINT ["/lexv2"]
