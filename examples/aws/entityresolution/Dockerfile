FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /entityresolution ./examples/aws/entityresolution
ENTRYPOINT ["/entityresolution"]
