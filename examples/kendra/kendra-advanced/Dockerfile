FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /kendra-advanced ./examples/kendra/kendra-advanced
ENTRYPOINT ["/kendra-advanced"]
