FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ecr ./examples/aws/ecr/ecr
ENTRYPOINT ["/ecr"]
