FROM golang:1.27-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /bin/server ./cmd/server

FROM alpine:3.24
COPY --from=build /bin/server /server
USER nobody
EXPOSE 8080
ENTRYPOINT ["/server"]
