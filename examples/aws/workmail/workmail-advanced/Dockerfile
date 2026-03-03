FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /workmail-advanced ./examples/aws/workmail/workmail-advanced
ENTRYPOINT ["/workmail-advanced"]
