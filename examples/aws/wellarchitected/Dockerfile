FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /wellarchitected ./examples/aws/wellarchitected
ENTRYPOINT ["/wellarchitected"]
