#
# STAGE 1: build static web files
#
FROM node:20-bookworm-slim AS frontend
WORKDIR /src

#
# install dependencies
COPY client/package*.json client/.npmrc ./
RUN npm install

#
# build client
COPY client/ .
RUN npm run build

#
# STAGE 2: build executable binary
#
FROM golang:1.24-bookworm AS builder
WORKDIR /app

COPY . .
RUN go get -v -t -d .; \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/neko_rooms cmd/neko_rooms/main.go

#
# STAGE 3: build a small image
#
# alpine rather than scratch: CA certificates for HTTPS (Discord), and the
# ssh client for docker engines reached over ssh://
FROM alpine:3.20
RUN apk add --no-cache ca-certificates openssh-client tzdata
COPY --from=builder /app/bin/neko_rooms /app/bin/neko_rooms
COPY --from=frontend /src/dist/ /var/www

ENV NEKO_ROOMS_BIND=:8080
ENV NEKO_ROOMS_ADMIN_STATIC=/var/www
# database with users, sessions, branding & settings
ENV NEKO_ROOMS_DATA_DIR=/data
VOLUME /data

EXPOSE 8080

ENTRYPOINT [ "/app/bin/neko_rooms" ]
CMD [ "serve" ]
