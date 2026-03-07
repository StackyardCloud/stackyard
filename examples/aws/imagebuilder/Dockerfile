FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /imagebuilder ./examples/aws/imagebuilder
ENTRYPOINT ["/imagebuilder"]
