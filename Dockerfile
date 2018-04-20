FROM nginx:1.13

LABEL maintainer="Stephen Afam-Osemene <stephenafamo@gmail.com>"

# ------------------------------------------
# install ping
# ------------------------------------------
RUN echo "deb http://ftp.debian.org/debian stretch-backports main" >> /etc/apt/sources.list \
	&& apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y inetutils-ping \
	openssl \
	# letsencrypt \
	certbot -t stretch-backports

# ------------------------------------------
# Copy custom commands and give appropriate premissions
# ------------------------------------------
COPY exec /docker/exec/
RUN mkdir /docker/config && touch /docker/config/config
RUN chmod 755 -R /docker/exec /docker/config

# ------------------------------------------
# Set add the location of executables to the path variable so they can be globally accessed
# ------------------------------------------
ENV PATH="/docker/exec:${PATH}"

# ------------------------------------------
# Copy custom nginx config and create config directories
# ------------------------------------------
COPY nginx.conf /etc/nginx/nginx.conf
RUN mkdir /etc/nginx/conf.d/http && mkdir /etc/nginx/conf.d/streams 

# ------------------------------------------
# copy our initilization file and set permissions
# ------------------------------------------
COPY init.sh /init.sh
RUN chmod 755 /init.sh

EXPOSE 443

CMD ["/init.sh"]