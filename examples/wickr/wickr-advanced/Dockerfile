FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /wickr-advanced ./examples/wickr/wickr-advanced
ENTRYPOINT ["/wickr-advanced"]
