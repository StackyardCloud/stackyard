FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /entityresolution-basic ./examples/aws/entityresolution/entityresolution-basic
ENTRYPOINT ["/entityresolution-basic"]
