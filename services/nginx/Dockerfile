FROM nginx:alpine

RUN rm /etc/nginx/conf.d/default.conf

COPY nginx_proxy.conf /etc/nginx/conf.d/nginx_proxy.conf

EXPOSE 8080