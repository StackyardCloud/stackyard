FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /step-functions-advanced ./examples/aws/step-functions/step-functions-advanced
ENTRYPOINT ["/step-functions-advanced"]
