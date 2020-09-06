FROM golang:1.14 AS builder

WORKDIR /usr/app

ADD . .

RUN make build







FROM nginx:1.19

LABEL maintainer="Stephen Afam-Osemene <me@stephenafamo.com>"
WORKDIR /usr/app

# ------------------------------------------
# install necessary packages
# ------------------------------------------
RUN apt-get update && apt-get install \
        --no-install-recommends --no-install-suggests -y \
        inetutils-ping \
        openssl \
        sqlite3 


# ------------------------------------------
# install certbot and its dns plugins
# the commented out plugins do not exist in the ppa
# ------------------------------------------
RUN apt-get install certbot --no-install-recommends --no-install-suggests -y \
        certbot \
        python3-certbot-dns-cloudflare \
        python3-certbot-dns-digitalocean \
        python3-certbot-dns-dnsimple \
        python3-certbot-dns-google \
        python3-certbot-dns-linode \
        python3-certbot-dns-ovh \
        python3-certbot-dns-rfc2136 \
        python3-certbot-dns-route53 
        # python3-certbot-dns-cloudxns \
        # python3-certbot-dns-dnsmadeeasy \
        # python3-certbot-dns-luadns \
        # python3-certbot-dns-nsone \

# ------------------------------------------
# Set the configuration directory
# ------------------------------------------
ENV CONFIG_DIR="/docker/config"

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
# Copy our warden executables
# ------------------------------------------
COPY --from=builder /usr/app/bin ./bin

EXPOSE 443

CMD ["./bin/warden"]
