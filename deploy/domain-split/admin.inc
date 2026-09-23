# imadmin.myad.top: include INSIDE the existing TLS server block.
# Replace the previous location / proxy; do not keep duplicate locations.
client_max_body_size 60m;
location / {
    include /www/server/panel/vhost/nginx/imgo-snippets/proxy.inc;
}
