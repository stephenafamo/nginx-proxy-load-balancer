#!/bin/bash

if [[ ! -z "$UPSTREAM" ]] && [[ ! -z "$DOMAIN" ]] ; then
	[[ ! -z "$UPSTREAM" ]] && container=$UPSTREAM 
	[[ ! -z "$DOMAIN" ]] && domain=$DOMAIN 
	[[ ! -z "$DIRECTORY" ]] && directory=$DIRECTORY 
	[[ ! -z "$SSL" ]] && ssl=$SSL 
	[[ ! -z "$SSL_SOURCE" ]] && ssl_source=$SSL_SOURCE 
	[[ ! -z "$HTTPS_ONLY" ]] && https_only=$HTTPS_ONLY 
	domain=$DOMAIN
	directory=$DIRECTORY
	ssl=$SSL
	ssl_source=$SSL_SOURCE
	https_only=$HTTPS_ONLY
	if [[ ! -z "$DIR" ]]; then
		directory="$DIR"
	fi
	add_config "$container" "$domain" "$directory" "$ssl" "$ssl_source" "$https_only"
fi

load_config

nginx -g "daemon off;"
