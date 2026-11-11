FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /global-accelerator ./examples/aws/global-accelerator
ENTRYPOINT ["/global-accelerator"]
