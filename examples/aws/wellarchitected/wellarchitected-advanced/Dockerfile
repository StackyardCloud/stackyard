FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /wellarchitected-advanced ./examples/aws/wellarchitected/wellarchitected-advanced
ENTRYPOINT ["/wellarchitected-advanced"]
