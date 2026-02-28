FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /wickr-basic ./examples/wickr/wickr-basic
ENTRYPOINT ["/wickr-basic"]
