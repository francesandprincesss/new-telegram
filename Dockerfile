FROM golang:1.27.3-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/grpc-chat-server ./cmd/server

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app
USER app

COPY --from=build /out/grpc-chat-server /usr/local/bin/grpc-chat-server

EXPOSE 50051
ENTRYPOINT ["grpc-chat-server"]
