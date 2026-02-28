FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ecr-advanced ./examples/ecr/ecr-advanced
ENTRYPOINT ["/ecr-advanced"]
