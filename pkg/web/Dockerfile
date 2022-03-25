# -- BUILD --
FROM node:14.17-alpine as build

WORKDIR /usr/src/app

COPY package* ./
COPY . .

RUN npm i
RUN npm run build

# -- RELEASE --
FROM nginx:stable-alpine as release

COPY --from=build /usr/src/app/build /usr/share/nginx/html
# copy .env.example as .env to the relase build
COPY --from=build /usr/src/app/.env.example /usr/share/nginx/html/.env

RUN apk add --update nodejs
RUN apk add --update npm
RUN npm i -g runtime-env-cra@0.2.0

WORKDIR /usr/share/nginx/html
EXPOSE 9090

# CMD ["/bin/sh", "-c", "runtime-env-cra && nginx -g \"daemon off;\""]
CMD ["/bin/sh", "-c", "nginx -g \"daemon off;\""]