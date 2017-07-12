FROM nginx:1.13

LABEL maintainer="Stephen Afam-Osemene <stephenafamo@gmail.com>"

# ------------------------------------------
# Copy custom commands and give appropriate premissions
# ------------------------------------------
COPY exec /docker/exec/
RUN chmod 755 -R /docker-exec

# ------------------------------------------
# Set add the location of executables to the path variable so they can be globally accessed
# ------------------------------------------
ENV PATH="/docker/exec:${PATH}"

# ------------------------------------------
# install ping
# ------------------------------------------
RUN apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y inetutils-ping \
	openssl \
	letsencrypt

# ------------------------------------------
# copy our initilization file and set permissions
# ------------------------------------------
COPY init.sh /init.sh
RUN chmod 755 /init.sh

CMD ["/init.sh"]