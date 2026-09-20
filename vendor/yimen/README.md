`jsbridge-mini.js` is the official Yimen App JavaScript SDK, downloaded from
https://www.yimenapp.com/doc/demo.cshtml?download-js-sdk=1 on 2026-09-21.

Original SDK SHA-256: `09fe681e0b196d2e6a9b04ed792bda4c249b4db88ea89c2ed7e621a9b7c643b0`.
The vendored copy only normalizes CRLF to LF; its SHA-256 is
`d77a03a6dc5c73dfc9f32978191327e9de0b0e33d43cfcac28bbb9b166e7dffd`.

The Yimen-only preparation script copies this SDK beside `index.html`. The
ordinary web H5 release does not load it.
