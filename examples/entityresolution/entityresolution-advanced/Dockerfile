FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /entityresolution-advanced ./examples/entityresolution/entityresolution-advanced
ENTRYPOINT ["/entityresolution-advanced"]
