FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /mediatailor-advanced ./examples/aws/mediatailor/mediatailor-advanced
ENTRYPOINT ["/mediatailor-advanced"]
