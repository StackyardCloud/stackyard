FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /clouddirectory-basic ./examples/clouddirectory/clouddirectory-basic
ENTRYPOINT ["/clouddirectory-basic"]
