# -- BUILD --
FROM node:14.17-alpine as build

WORKDIR /usr/src/app

COPY package* ./
COPY . .

RUN npm i
RUN npm run build

# -- RELEASE --
FROM openresty/openresty:alpine as release

COPY --from=build /usr/src/app/build /usr/share/nginx/html
# copy .env.example as .env to the relase build
COPY --from=build /usr/src/app/.env.example /usr/share/nginx/html/.env

# Copy lua modules
COPY nginx/lua/ /usr/local/openresty/nginx/lua/
COPY nginx/nginx.conf /usr/local/openresty/nginx/conf/nginx.conf

RUN apk add --update nodejs
RUN apk add --update npm
RUN npm i -g runtime-env-cra@0.2.0

WORKDIR /usr/share/nginx/html
EXPOSE 9090

# CMD ["/bin/sh", "-c", "runtime-env-cra && /usr/local/openresty/bin/openresty -g \"daemon off;\""]

CMD ["/usr/local/openresty/bin/openresty", "-g", "daemon off;"]

STOPSIGNAL SIGQUIT
