FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /wickr ./examples/aws/wickr
ENTRYPOINT ["/wickr"]
