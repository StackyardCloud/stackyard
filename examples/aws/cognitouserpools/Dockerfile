FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /cognitouserpools ./examples/aws/cognitouserpools
ENTRYPOINT ["/cognitouserpools"]
