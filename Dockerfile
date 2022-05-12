FROM alpine:latest

COPY ./ingress-default-backend /app/ingress-default-backend
COPY ./templates/ /app/templates/

WORKDIR /app

RUN apk upgrade \
&& addgroup -g 30000 -S app \
&& adduser -u 30000 -D -S -G app app

USER 30000

ENTRYPOINT [ "/app/ingress-default-backend" ]