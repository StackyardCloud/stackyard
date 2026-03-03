FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /evs ./examples/aws/evs/evs
ENTRYPOINT ["/evs"]
