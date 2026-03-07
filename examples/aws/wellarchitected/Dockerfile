FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /wellarchitected ./examples/aws/wellarchitected
ENTRYPOINT ["/wellarchitected"]
