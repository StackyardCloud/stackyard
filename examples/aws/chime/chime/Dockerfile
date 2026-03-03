FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /chime ./examples/aws/chime/chime
ENTRYPOINT ["/chime"]
