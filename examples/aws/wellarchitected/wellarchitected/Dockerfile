FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /wellarchitected ./examples/aws/wellarchitected/wellarchitected
ENTRYPOINT ["/wellarchitected"]
