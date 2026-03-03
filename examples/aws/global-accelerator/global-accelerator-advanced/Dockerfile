FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /global-accelerator-advanced ./examples/aws/global-accelerator/global-accelerator-advanced
ENTRYPOINT ["/global-accelerator-advanced"]
