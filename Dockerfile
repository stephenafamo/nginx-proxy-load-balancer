FROM nginx:1.13
MAINTAINER Stephen Afam-Osemene <stephenafamo@gmail.com>
COPY init.sh /init.sh
RUN chmod 755 /init.sh
CMD ["/init.sh"]