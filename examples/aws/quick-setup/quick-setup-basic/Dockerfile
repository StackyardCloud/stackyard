FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /quick-setup-basic ./examples/aws/quick-setup/quick-setup-basic
ENTRYPOINT ["/quick-setup-basic"]
