FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /global-accelerator ./examples/aws/global-accelerator/global-accelerator
ENTRYPOINT ["/global-accelerator"]
