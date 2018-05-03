FROM nginx:1.13

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
# Set the location of executables to the path variable so they can be globally accessed
# ------------------------------------------
ENV PATH="/docker/exec:${PATH}"

# ------------------------------------------
# Set the configuration directory
# ------------------------------------------
ENV CONFIG_DIR="/docker/config"

# ------------------------------------------
# Set the reload duration
# ------------------------------------------
ENV CONFIG_VALIDITY="604800"

# ------------------------------------------
# Set the configuration directory
# ------------------------------------------
ENV CONFIG_RELOAD_TIME="5s"

# ------------------------------------------
# Set the sqlite db file
# ------------------------------------------
ENV CONFIG_DB="/docker/db/files.db"

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

# ------------------------------------------
# Copy custom commands and give appropriate premissions
# ------------------------------------------
COPY exec /docker/exec/
RUN mkdir /docker/config && touch /docker/config/config
RUN chmod 755 -R /docker/exec /docker/config

EXPOSE 443

CMD ["/init.sh"]