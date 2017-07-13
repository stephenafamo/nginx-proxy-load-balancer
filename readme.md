# **Docker Nginx Auto Proxy**
This docker image automatic proxies requests to your docker containers

## Usage

## Simple configuration

You can use the environmental variables `CONTAINER`, `UPSTREAM`, `DIRECTORY`, `SSL`, `SSL_SOURCE` and `HTTPS_ONLY` to set a single line of the config while bringing up the container

    docker run --name nginx-proxy \
    -e UPSTREAM="awesome_blog" \
    -e DOMAIN="stephenafamo.com" \
    -e DIRECTORY="blog" \
    -e SSL="1" \
    -e SSL_SOURCE="letsencrypt" \
    -e HTTPS_ONLY="1" \
    -p 80:80 \
    -d stephenafamo/docker-nginx-auto-proxy

## Full configuration

The container reads a configuration file `/docker/config/config`
To easily manage all proxies, you should mount your own configuration file.
`-v /path/to/my/config.txt:/docker/config/config`
The syntax is as follows.

1. **Single line configuration**
    `"upstream"="domains_to_proxy"="directory_to_proxy"="ssl"="ssl_source"="https_only"`
    1. Neither the domain or upstream address should include the scheme `http://`
    2. `upstream` must be reachable or the config will not be generated
    3. `domains_to_proxy` can be multiple, but should be seperated by spaces
    4. `directory` is the optional path to to be proxies. For example, if you'd like to proxy only `example.com/blog`, the `blog` will be the directory
    5. `ssl` can be enable by setting the parameter to `1`
    6. `ssl_source` for now, only letsencrypt is supported. Certificates will be generated automatically. Soon, mannual configuration will be supported. To be able to re-use the generated certificates, you should mount your `/etc/letsencrypt` folder into the container `-v /etc/letsencrypt:/etc/letsencrypt`. THis only works if `ssl` is `1`.
    7. `https_only` If this is set to `1`, then all `http` requests will be redirected to `https`
2. **Block Configuration**
    Syntax

        myblog==
        ==UPSTREAM="stephenafamo.com"
        ==DOMAIN=business.hotels.ng.dev
        ==DIRECTORY=blog
        ==SSL=1
        ==SSL_SOURCE="letsencrypt"
        ==SSL_MODE="any"
    The same rules apply.

## Additional commands 

The following commands are available through the contianer.

1. **_active_domains_**: Will list out the domains that have been configured
2. **_add_config_** _upstream domain directory ssl ssl_source https_only_: Will add a domain to the config file
3. **_load_config_**: Will re-generate configuration files and reload nginx

## Let's Encrypt

If set up correctly, the container will attempt to get a new certificate if there was none, or renew the certificate.
The normal `letsencrypt renew` command may fail. Instead, to renew certificates, run the `load_config` command and certificate renewal will be attempted during the process. 
You should set up a cron to do this automatically e.g `30 2 * * 1 docker exec nginx load_config >> /var/log/nginx-reload.log`

## Roadmap

* Load balancing with multiple containers
* ~~Automatic SSL support with let's encrypt~~ **DONE**

Please inform me of any issues, Pull requests are appreciated.
