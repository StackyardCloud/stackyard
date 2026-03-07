FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /amplifyuibuilder ./examples/aws/amplifyuibuilder
ENTRYPOINT ["/amplifyuibuilder"]
