FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /quick-setup ./examples/aws/quick-setup/quick-setup
ENTRYPOINT ["/quick-setup"]
