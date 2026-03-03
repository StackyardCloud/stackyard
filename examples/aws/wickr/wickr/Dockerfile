FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /wickr ./examples/aws/wickr/wickr
ENTRYPOINT ["/wickr"]
