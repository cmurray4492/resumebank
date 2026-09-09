# Multi-stage build: compile a static Go binary, then run it in a small
# runtime image. Templates and static assets are embedded into the binary
# at build time (see web/assets.go and internal/render), so the final
# image needs nothing but the compiled binaries.
FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server
RUN CGO_ENABLED=0 go build -o /out/migrate ./cmd/migrate
RUN CGO_ENABLED=0 go build -o /out/createadmin ./cmd/createadmin

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/server /app/server
COPY --from=build /out/migrate /app/migrate
COPY --from=build /out/createadmin /app/createadmin

EXPOSE 8080
CMD ["/app/server"]
