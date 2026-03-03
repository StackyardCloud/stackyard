FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /imagebuilder-basic ./examples/aws/imagebuilder/imagebuilder-basic
ENTRYPOINT ["/imagebuilder-basic"]
