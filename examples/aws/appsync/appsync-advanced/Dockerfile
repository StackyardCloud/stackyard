FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /appsync-advanced ./examples/aws/appsync/appsync-advanced
ENTRYPOINT ["/appsync-advanced"]
