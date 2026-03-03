FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /artifact-advanced ./examples/aws/artifact/artifact-advanced
ENTRYPOINT ["/artifact-advanced"]
