package internal

import (
	"text/template"
)

func GetTemplates() (*template.Template, error) {
	t := template.New("configs")

	err := parseHttp(t)
	if err != nil {
		panic(err)
	}

	err = parseStream(t)
	if err != nil {
		panic(err)
	}

	err = parseHttps(t)
	if err != nil {
		panic(err)
	}

	err = parseHttptoHttps(t)
	if err != nil {
		panic(err)
	}

	return t, nil
}

func parseHttp(t *template.Template) error {
	nt := t.New("httpBase")
	_, err := nt.Parse(`
        {{- if .Location -}}
        upstream {{.Unique}} {
            {{range .Upstream }}
            server {{.Address}}{{range .Parameters}} {{.}}{{end}};
            {{- end}}
            {{range $i, $x := $.UpstreamOptions }}
            {{ $i }} {{ $x }};
            {{- end}}
        }
        {{- end}}

        {{range $i, $x := $.Locations }}
        upstream {{$.Unique}}-{{$i}} {
            {{range $x.Upstream }}
            server {{.Address}}{{range .Parameters}} {{.}}{{end}};
            {{- end}}
            {{range $j, $y := $x.UpstreamOptions }}
            {{ $j }} {{ $y }};
            {{- end}}
        }
        {{- end}}

        server {
            listen 80;
            listen [::]:80;
            server_name {{- range .Domains}} {{.}}{{end}};
            {{range $i, $x := $.ServerOptions }}
            {{ $i }} {{ $x }};
            {{- end}}

            location ^~ /.well-known/acme-challenge {
                default_type "text/plain";
                root /docker/challenge/{{$.Unique}};
                allow all;
            }

            {{if .Location -}}
            location {{.Location}} {
                proxy_pass http://{{.Unique}};

                proxy_set_header Host $http_host;
                proxy_set_header X-Real-IP $remote_addr;
                proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
                proxy_set_header X-Forwarded-Proto $scheme;

                {{range $i, $x := $.LocationOptions }}
                {{ $i }} {{ $x }};
                {{- end}}
            }
            {{- end}}

            {{range $i, $x := $.Locations }}
            location {{$x.Match}} {
                proxy_pass http://{{$.Unique}}-{{$i}};

                proxy_set_header Host $http_host;
                proxy_set_header X-Real-IP $remote_addr;
                proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
                proxy_set_header X-Forwarded-Proto $scheme;

                {{range $j, $y := $x.Options -}}
                {{ $j }} {{ $y }};
                {{- end}}
            }
            {{end}}
        }
    `)
	if err != nil {
		return err
	}

	return nil
}

func parseStream(t *template.Template) error {
	nt := t.New("streams")
	_, err := nt.Parse(`
        upstream {{.Unique}}  {
            {{range .Upstream }}
            server {{.Address}}{{range .Parameters}} {{.}}{{end}};
            {{- end}}
            {{range $i, $x := $.UpstreamOptions }}
            {{ $i }} {{ $x }};
            {{- end}}
        }

        server {
            listen 80;
            listen [::]:80;

            proxy_pass {{.Unique}};
            {{range $i, $x := $.ServerOptions }}
            {{ $i }} {{ $x }};
            {{- end}}
        }
    `)
	if err != nil {
		return err
	}

	return nil
}

func parseHttps(t *template.Template) error {
	nt := t.New("https")
	_, err := nt.Parse(`
        server {
            listen 4343 ssl http2;
            listen [::]:4343 ssl http2;
            server_name {{- range .Domains}} {{.}}{{end}};
            {{range $i, $x := $.ServerOptions }}
            {{ $i }} {{ $x }};
            {{- end}}

            ssl_certificate {{ .CertPath }};
            ssl_certificate_key {{ .KeyPath }};
            ssl_session_cache shared:SSL:10m;
            ssl_session_timeout 5m;
            ssl_protocols TLSv1 TLSv1.1 TLSv1.2;
            ssl_prefer_server_ciphers on;

            ssl_ciphers 'ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-AES128-SHA256:ECDHE-RSA-AES128-SHA256:ECDHE-ECDSA-AES128-SHA:ECDHE-RSA-AES256-SHA384:ECDHE-RSA-AES128-SHA:ECDHE-ECDSA-AES256-SHA384:ECDHE-ECDSA-AES256-SHA:ECDHE-RSA-AES256-SHA:DHE-RSA-AES128-SHA256:DHE-RSA-AES128-SHA:DHE-RSA-AES256-SHA256:DHE-RSA-AES256-SHA:ECDHE-ECDSA-DES-CBC3-SHA:ECDHE-RSA-DES-CBC3-SHA:EDH-RSA-DES-CBC3-SHA:AES128-GCM-SHA256:AES256-GCM-SHA384:AES128-SHA256:AES256-SHA256:AES128-SHA:AES256-SHA:DES-CBC3-SHA:!DSS';
            ssl_stapling on;
            ssl_stapling_verify on;
            add_header Strict-Transport-Security max-age=15768000;

            location ^~ /.well-known/acme-challenge {
                default_type "text/plain";
                root /docker/challenge/{{$.Unique}};
                allow all;
            }

            {{if .Location -}}
            location {{.Location}} {
                proxy_pass http://{{.Unique}};

                proxy_set_header Host $http_host;
                proxy_set_header X-Real-IP $remote_addr;
                proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
                proxy_set_header X-Forwarded-Proto $scheme;

                {{range $i, $x := $.LocationOptions }}
                {{ $i }} {{ $x }};
                {{- end}}
            }
            {{- end}}

            {{range $i, $x := $.Locations }}
            location {{$x.Match}} {
                proxy_pass http://{{$.Unique}}-{{$i}};

                proxy_set_header Host $http_host;
                proxy_set_header X-Real-IP $remote_addr;
                proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
                proxy_set_header X-Forwarded-Proto $scheme;

                {{range $j, $y := $x.Options -}}
                {{ $j }} {{ $y }};
                {{- end}}
            }
            {{- end}}

        }
    `)
	if err != nil {
		return err
	}

	return nil
}

func parseHttptoHttps(t *template.Template) error {
	nt := t.New("httptoHttps")
	_, err := nt.Parse(`
        {{if .Location -}}
        upstream {{.Unique}} {
            {{range .Upstream }}
            server {{.Address}}{{range .Parameters}} {{.}}{{end}};
            {{- end}}
            {{range $i, $x := $.UpstreamOptions }}
            {{ $i }} {{ $x }};
            {{- end}}
        }
        {{- end}}

        {{range $i, $x := $.Locations }}
        upstream {{$.Unique}}-{{$i}} {
            {{range $x.Upstream }}
            server {{.Address}}{{range .Parameters}} {{.}}{{end}};
            {{- end}}
            {{range $j, $y := $x.UpstreamOptions }}
            {{ $j }} {{ $y }};
            {{- end}}
        }
        {{- end}}

        server {
            listen 80;
            listen [::]:80;
            server_name {{- range .Domains}} {{.}}{{end}};
            {{range $i, $x := $.ServerOptions }}
            {{ $i }} {{ $x }};
            {{- end}}

            location ^~ /.well-known/acme-challenge {
                default_type "text/plain";
                root /docker/challenge/{{$.Unique}};
                allow all;
            }

            {{if .Location -}}
            location {{.Location}} {
                return 301 https://$server_name$request_uri;
				expires 1h;
            }
            {{- end}}
            {{range $i, $x := $.Locations }}
            location {{$x.Match}} {
                return 301 https://$server_name$request_uri;
				expires 1h;
            }
            {{- end}}
        }
    `)
	if err != nil {
		return err
	}

	return nil
}
