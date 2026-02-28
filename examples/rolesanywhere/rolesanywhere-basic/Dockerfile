FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /rolesanywhere-basic ./examples/rolesanywhere/rolesanywhere-basic
CMD ["/rolesanywhere-basic"]
