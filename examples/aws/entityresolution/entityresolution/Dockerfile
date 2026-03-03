FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /entityresolution ./examples/aws/entityresolution/entityresolution
ENTRYPOINT ["/entityresolution"]
