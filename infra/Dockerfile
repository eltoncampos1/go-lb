FROM golang:1.25-alpine AS build

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN go build -o lb ./cmd/lb

FROM alpine:3.20

WORKDIR /app

COPY --from=build /app/lb .

EXPOSE 3030

CMD ["./lb"]

