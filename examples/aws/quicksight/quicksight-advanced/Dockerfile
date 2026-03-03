FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /quicksight-advanced ./examples/aws/quicksight/quicksight-advanced
ENTRYPOINT ["/quicksight-advanced"]
