FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cognitouserpools-advanced ./examples/cognitouserpools/cognitouserpools-advanced
ENTRYPOINT ["/cognitouserpools-advanced"]
