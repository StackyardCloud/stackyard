FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /amplifyuibuilder-basic ./examples/aws/amplifyuibuilder/amplifyuibuilder-basic
ENTRYPOINT ["/amplifyuibuilder-basic"]
