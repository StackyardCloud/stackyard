FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /amplifyuibuilder ./examples/aws/amplifyuibuilder/amplifyuibuilder
ENTRYPOINT ["/amplifyuibuilder"]
