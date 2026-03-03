FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /lexv2-advanced ./examples/aws/lexv2/lexv2-advanced
ENTRYPOINT ["/lexv2-advanced"]
