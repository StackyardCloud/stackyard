FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cognitouserpools-basic ./examples/aws/cognitouserpools/cognitouserpools-basic
ENTRYPOINT ["/cognitouserpools-basic"]
