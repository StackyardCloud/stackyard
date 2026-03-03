FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /evs-advanced ./examples/aws/evs/evs-advanced
ENTRYPOINT ["/evs-advanced"]
