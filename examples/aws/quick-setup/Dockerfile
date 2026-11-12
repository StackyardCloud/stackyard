FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /quick-setup ./examples/aws/quick-setup
ENTRYPOINT ["/quick-setup"]
