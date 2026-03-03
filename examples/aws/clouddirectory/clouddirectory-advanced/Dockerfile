FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /clouddirectory-advanced ./examples/aws/clouddirectory/clouddirectory-advanced
ENTRYPOINT ["/clouddirectory-advanced"]
