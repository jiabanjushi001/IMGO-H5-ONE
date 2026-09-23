# imgo.myad.top: include INSIDE the existing TLS server block.
# Replace the previous location / proxy; do not keep duplicate locations.
# Keep certificate / ACME configuration maintained by BaoTa.
client_max_body_size 60m;
location = / { return 404; }
location ~* ^/index\.html(?:/|$) { return 404; }
location / {
    include /www/server/panel/vhost/nginx/imgo-snippets/proxy.inc;
}
