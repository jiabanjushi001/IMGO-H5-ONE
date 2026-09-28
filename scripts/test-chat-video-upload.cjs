const assert = require('assert/strict');
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const {execFileSync} = require('child_process');

const root = path.resolve(__dirname, '..');
const source = fs.readFileSync(path.join(root, 'frontend/config-state.js'), 'utf8');
const context = {Number, Object, Array, Set, String};
vm.runInNewContext(source, context);

assert.equal(context.imgoUploadLimitMB({type: 'video'}, {size: 50}), 200);
assert.equal(context.imgoUploadLimitMB({type: 'video'}, {size: 50, videoSize: 120}), 120);
assert.equal(context.imgoUploadLimitMB({type: 'file'}, {size: 50, videoSize: 200}), 50);

execFileSync(process.execPath, ['scripts/build-maintenance.cjs'], {cwd: root, stdio: 'inherit'});
const app = fs.readFileSync(path.join(root, 'public/assets/js/app.85372e4e.js'), 'utf8');
assert.ok(app.includes('imgoUploadLimitMB(t,this.globalConfig.fileUpload)'), 'video-aware upload limit must be built into the chat bundle');
assert.ok(app.includes('上传的内容不能大于'), 'upload error copy must be corrected');

console.log('Chat video upload tests passed');
