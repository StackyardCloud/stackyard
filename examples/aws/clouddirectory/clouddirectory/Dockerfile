FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /clouddirectory ./examples/aws/clouddirectory/clouddirectory
ENTRYPOINT ["/clouddirectory"]
