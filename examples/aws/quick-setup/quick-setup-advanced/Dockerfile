FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /quick-setup-advanced ./examples/aws/quick-setup/quick-setup-advanced
ENTRYPOINT ["/quick-setup-advanced"]
