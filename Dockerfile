FROM golang:1.12 AS builder
ADD . /usr/app
WORKDIR /usr/app
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -a -o /warden .

FROM nginx:1.15

LABEL maintainer="Stephen Afam-Osemene <stephenafamo@gmail.com>"

# ------------------------------------------
# install ping
# ------------------------------------------
RUN echo "deb http://ftp.debian.org/debian stretch-backports main" >> /etc/apt/sources.list \
	&& apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y inetutils-ping \
	openssl \
	sqlite3 \
	certbot -t stretch-backports

# ------------------------------------------
# Set the configuration directory
# ------------------------------------------
ENV CONFIG_DIR="/docker/config"

# ------------------------------------------
# Set the validity duration
# ------------------------------------------
ENV CONFIG_VALIDITY="30d"

# ------------------------------------------
# Set the reload duration
# ------------------------------------------
ENV CONFIG_RELOAD_TIME="5s"

# ------------------------------------------
# Copy custom nginx config and create config directories
# ------------------------------------------
COPY ./config/nginx.conf /etc/nginx/nginx.conf
RUN mkdir -p /docker/config /etc/nginx/conf.d/http /etc/nginx/conf.d/streams 

# ------------------------------------------
# Remove symlink for NGINX logs
# ------------------------------------------
RUN rm -rf /var/log/nginx/*.log && touch /var/log/nginx/access.log /var/log/nginx/error.log

# ------------------------------------------
# Copy our warden executable
# ------------------------------------------
COPY --from=builder /warden /usr/bin

EXPOSE 443

CMD ["warden"]