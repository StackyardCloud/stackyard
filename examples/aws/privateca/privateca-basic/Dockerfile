FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /privateca-basic ./examples/aws/privateca/privateca-basic
ENTRYPOINT ["/privateca-basic"]
