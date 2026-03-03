FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /artifact-basic ./examples/aws/artifact/artifact-basic
ENTRYPOINT ["/artifact-basic"]
