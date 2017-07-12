#!/bin/bash

if [[ ! -z "$UPSTREAM" ]] && [[ ! -z "$DOMAIN" ]] ; then
	container=$UPSTREAM
	domain=$DOMAIN
	directory=$DIRECTORY
	ssl=$SSL
	ssl_source=$SSL_SOURCE
	https_only=$HTTPS_ONLY
	if [[ ! -z "$DIR" ]]; then
		directory="$DIR"
	fi
	add_config $container $domain $directory $ssl $ssl_source $https_only
fi

load_config

nginx -g "daemon off;"
