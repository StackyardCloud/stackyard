FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /evs-basic ./examples/evs/evs-basic
ENTRYPOINT ["/evs-basic"]
