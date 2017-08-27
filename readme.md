# **Docker Nginx Auto Proxy**
This docker image automatic proxies requests to your docker containers

## Usage

## Configuration

The container reads a configuration file `/docker/config/config`
To easily manage all proxies, you should mount your own configuration file.
`-v /path/to/my/config.txt:/docker/config/config`
The syntax is as follows.

    Syntax

        "myblog"
        "UPSTREAM"main.stephenafamo.com|1st.stephenafamo.com weight=3|2nd.stpehenafamo.com max_fails=3 fail_timeout=30s"
        "UPSTREAM_OPTIONS"ip_hash|keep_alive 32"
        "DOMAIN"stephenafamo.com"
        "DIRECTORY"blog"
        "SSL"1"
        "SSL_SOURCE"letsencrypt"
        "SSL_MODE"any"
        "myblog"


1. A block of configuration should be started and ended by the configuration name. This name should be unique. In the example above, the configuration name is `myblog`
2. Neither the domain or upstream address should include the scheme `http://`
3. `UPSTREAM` must be reachable or the config will not be generated.
4. For load balancing, you can add multiple `UPSTREAM` addresses. Separate them with pipes.
5. You can add any extra parameters at the end of a single upstream server. [Read this](http://nginx.org/en/docs/http/ngx_http_upstream_module.html#server).
6. The `UPSTREAM_OPTIONS` are not required. Use only if you need to add extra directives to the upstream block for fine tuning. Separate directives with pipes. [Read this](http://nginx.org/en/docs/http/ngx_http_upstream_module.html).
7. `DOMAIN` can be multiple, but should be seperated by spaces
8. `DIRECTORY` is the optional path to to be proxies. For example, if you'd like to proxy only `example.com/blog`, the `blog` will be the directory
9. `SSL` can be enable by setting the parameter to `1`
10. `SSL_SOURCE` for now, only letsencrypt is supported. Certificates will be generated automatically. Soon, mannual configuration will be supported. To be able to re-use the generated certificates, you should mount your `/etc/letsencrypt` folder into the container `-v /etc/letsencrypt:/etc/letsencrypt`. THis only works if `SSL` is `1`.
11. `HTTPS_ONLY` If this is set to `1`, then all `http` requests will be redirected to `https`

## Additional commands 

The following commands are available through the contianer.

1. **_active_domains_**: Will list out the domains that have been configured
3. **_load_config_**: Will re-generate configuration files and reload nginx

## Let's Encrypt

If set up correctly, the container will attempt to get a new certificate if there was none, or renew the certificate.
The normal `letsencrypt renew` command may fail. Instead, to renew certificates, run the `load_config` command and certificate renewal will be attempted during the process. 
You should set up a cron to do this automatically e.g `30 2 * * 1 docker exec nginx load_config >> /var/log/nginx-reload.log`

## Roadmap

* ~~Load balancing with multiple containers~~ **DONE**
* ~~Automatic SSL support with let's encrypt~~ **DONE**
* Allow custom ssl certificate configuration.

Please inform me of any issues, Pull requests are appreciated.
