FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /step-functions-basic ./examples/step-functions/step-functions-basic
ENTRYPOINT ["/step-functions-basic"]
