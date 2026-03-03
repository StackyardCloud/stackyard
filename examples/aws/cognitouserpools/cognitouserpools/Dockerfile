FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cognitouserpools ./examples/aws/cognitouserpools/cognitouserpools
ENTRYPOINT ["/cognitouserpools"]
