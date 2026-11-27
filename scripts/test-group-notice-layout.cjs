const assert = require('assert');
const fs = require('fs');
const path = require('path');

const root = path.resolve(__dirname, '..');
const read = (file) => fs.readFileSync(path.join(root, file), 'utf8');

const chatCss = read('frontend/chat-design.css');
const messageCss = read('frontend/message-panel.css');

// Only the outer Lemon IM shell may fill the chat panel. The group member
// sidebar is itself a nested `.lemon-wrapper` with a fixed 180px width. A
// descendant selector here overrides that width with `100% !important`, so a
// long announcement can expand the sidebar and squeeze the composer to 0px.
assert.ok(
  chatCss.includes('.imgo-chat-design > .lemon-wrapper { width:100% !important;'),
  'standalone chat must size only its root Lemon wrapper'
);
assert.ok(
  messageCss.includes('.sideMenu-message .chat-box > .lemon-wrapper { width:100% !important;'),
  'management quick chat must size only its root Lemon wrapper'
);
assert.ok(
  !chatCss.includes('.imgo-chat-design .lemon-wrapper { width:100% !important;'),
  'standalone chat must not override nested group sidebar width'
);
assert.ok(
  !messageCss.includes('.sideMenu-message .lemon-wrapper { width:100% !important;'),
  'management quick chat must not override nested group sidebar width'
);

const builtCss = read('public/assets/css/imgo-maintenance.css');
assert.ok(builtCss.includes('.imgo-chat-design > .lemon-wrapper { width:100% !important;'));
assert.ok(builtCss.includes('.sideMenu-message .chat-box > .lemon-wrapper { width:100% !important;'));

console.log('group notice layout tests passed');
