# -- BUILD --
FROM node:14.17-alpine as build

WORKDIR /usr/src/app

COPY package* ./
COPY . .

RUN npm i
RUN npm run build

# -- RELEASE --
FROM caddy:alpine as release

COPY --from=build /usr/src/app/build /usr/share/caddy

WORKDIR /usr/share/caddy
EXPOSE 9090

CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile", "--watch"]