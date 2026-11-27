const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

const component = {methods: {}, mounted() {}, beforeDestroy() {}};
let errorHandler = null;
const document = {
  addEventListener(type, handler, capture) {
    if (type === 'error' && capture) errorHandler = handler;
  },
  removeEventListener(type, handler, capture) {
    assert.equal(type, 'error');
    assert.equal(handler, errorHandler);
    assert.equal(capture, true);
  }
};
const windowObject = {
  location: {
    href: 'http://127.0.0.1:8088/#/chat',
    origin: 'http://127.0.0.1:8088'
  }
};
vm.runInNewContext(fs.readFileSync('frontend/media-auth-recovery.js', 'utf8'), {
  component,
  document,
  window: windowObject,
  URL,
  Promise,
  Number,
  Array,
  Date
});

async function main() {
  let syncCalls = 0;
  const instance = {
    $api: {imApi: {getContactsAPI() {
      syncCalls += 1;
      return Promise.resolve({code: 0, data: []});
    }}}
  };
  for (const [name, method] of Object.entries(component.methods)) instance[name] = method.bind(instance);
  component.mounted.call(instance);
  assert.equal(typeof errorHandler, 'function', 'chat must listen for protected image failures');
  const attrs = new Map([['src', '/storage/image/a.png']]);
  const image = {
    tagName: 'IMG',
    currentSrc: 'http://127.0.0.1:8088/storage/image/a.png',
    src: '',
    getAttribute(name) { return attrs.get(name) || null; },
    setAttribute(name, value) { attrs.set(name, value); }
  };
  errorHandler({target: image});
  await instance._imgoMediaCookieSyncPromise;
  await Promise.resolve();
  assert.equal(syncCalls, 1, 'failed protected image must refresh the media cookie');
  assert.match(image.src, /_imgo_media_retry=/, 'image must retry with a cache-busting URL after auth sync');
  errorHandler({target: image});
  assert.equal(syncCalls, 1, 'the same broken image must not enter a retry loop');
  component.beforeDestroy.call(instance);
  console.log('media auth recovery checks passed');
}

main().catch(error => {
  console.error(error);
  process.exitCode = 1;
});
