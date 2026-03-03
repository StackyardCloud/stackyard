FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /controlcatalog-basic ./examples/aws/controlcatalog/controlcatalog-basic
ENTRYPOINT ["/controlcatalog-basic"]
