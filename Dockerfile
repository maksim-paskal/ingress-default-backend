FROM golang:1.14 as build

COPY *.go /usr/src/ingress-default-backend/
COPY go.* /usr/src/ingress-default-backend/

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

RUN cd /usr/src/ingress-default-backend \
  && go mod download \
  && go mod verify \
  && go build -v -o ingress-default-backend -ldflags "-X main.buildTime=$(date +"%Y%m%d%H%M%S")"

FROM alpine:latest

COPY --from=build /usr/src/ingress-default-backend/ingress-default-backend /app/ingress-default-backend
COPY ./templates/ /app/templates/

WORKDIR /app
CMD /app/ingress-default-backend