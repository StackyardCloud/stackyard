FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /imagebuilder ./examples/aws/imagebuilder/imagebuilder
ENTRYPOINT ["/imagebuilder"]
