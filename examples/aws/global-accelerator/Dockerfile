FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /global-accelerator ./examples/aws/global-accelerator
ENTRYPOINT ["/global-accelerator"]
