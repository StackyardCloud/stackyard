FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /global-accelerator-basic ./examples/aws/global-accelerator/global-accelerator-basic
ENTRYPOINT ["/global-accelerator-basic"]
