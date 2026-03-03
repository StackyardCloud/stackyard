FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ecr-basic ./examples/aws/ecr/ecr-basic
ENTRYPOINT ["/ecr-basic"]
