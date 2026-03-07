FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /quick-setup ./examples/aws/quick-setup
ENTRYPOINT ["/quick-setup"]
