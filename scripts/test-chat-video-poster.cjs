const fs = require('fs');

const source = fs.readFileSync('frontend/chat-video-poster.js', 'utf8');
const app = fs.readFileSync('public/assets/js/app.85372e4e.js', 'utf8');

for (const expected of [
  "video.addEventListener('loadeddata'",
  'context.drawImage(video, 0, 0, canvas.width, canvas.height)',
  "canvas.toBlob(function (blob)",
  "Object.defineProperty(file, '__imgoPoster'"
]) {
  if (!source.includes(expected)) throw new Error(`video poster source is missing: ${expected}`);
}
if (!app.includes('IMGO_FRIEND_CHAT_BEGIN')) throw new Error('chat adapter is not installed');
if (!app.includes('__imgoPoster')) throw new Error('video poster extractor is not installed');
if (!app.includes('n.append("poster",i.__imgoPoster,"video-cover.jpg")')) {
  throw new Error('video poster is not attached to the upload request');
}
console.log('Chat video first-frame poster checks passed');
