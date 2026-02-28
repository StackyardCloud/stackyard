FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /quicksight-basic ./examples/quicksight/quicksight-basic
ENTRYPOINT ["/quicksight-basic"]
