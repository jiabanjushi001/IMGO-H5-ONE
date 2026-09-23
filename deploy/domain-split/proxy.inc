# Shared settings; include inside a location block.
proxy_pass http://127.0.0.1:8088;
proxy_http_version 1.1;
proxy_set_header Host $host;
proxy_set_header X-Forwarded-For $remote_addr;
proxy_set_header X-Forwarded-Proto $scheme;
proxy_set_header Upgrade $http_upgrade;
# No global map needed when pasting into a BaoTa site configuration.
proxy_set_header Connection "upgrade";
proxy_read_timeout 120s;
proxy_send_timeout 120s;
proxy_buffering off;
