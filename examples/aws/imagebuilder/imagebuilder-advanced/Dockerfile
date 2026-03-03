FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /imagebuilder-advanced ./examples/aws/imagebuilder/imagebuilder-advanced
ENTRYPOINT ["/imagebuilder-advanced"]
