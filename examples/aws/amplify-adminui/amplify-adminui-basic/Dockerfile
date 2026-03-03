FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /amplify-adminui-basic ./examples/aws/amplify-adminui/amplify-adminui-basic
ENTRYPOINT ["/amplify-adminui-basic"]
