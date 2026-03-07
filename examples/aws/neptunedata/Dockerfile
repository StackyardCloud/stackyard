FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /neptunedata ./examples/aws/neptunedata
ENTRYPOINT ["/neptunedata"]
