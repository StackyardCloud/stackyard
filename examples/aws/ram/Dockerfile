FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /ram ./examples/aws/ram
ENTRYPOINT ["/ram"]
