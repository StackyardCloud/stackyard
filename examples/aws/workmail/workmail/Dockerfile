FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /workmail ./examples/aws/workmail/workmail
ENTRYPOINT ["/workmail"]
