const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const {execFileSync} = require('node:child_process');

const root = path.resolve(__dirname, '..');
const build = () => execFileSync(process.execPath, ['scripts/build-maintenance.cjs'], {cwd: root, stdio: 'pipe'});

build();
build();

const app = fs.readFileSync(path.join(root, 'public/assets/js/app.85372e4e.js'), 'utf8');
assert.equal((app.match(/IMGO_ADMIN_DIRECT_UPLOAD_BEGIN/g) || []).length, 1);
assert.match(app, /prepareDirect/);
assert.match(app, /completeDirect/);
assert.match(app, /sendFileAPI=t=>imgoAdminUpload/);
assert.match(app, /uploadAvatar:t=>imgoAdminUpload/);

const chunk = app.match(/789:"(imgo[a-f0-9]+)"/);
assert.ok(chunk, 'settings chunk hash missing');
const settings = fs.readFileSync(path.join(root, `public/assets/js/789.${chunk[1]}.js`), 'utf8');
assert.match(settings, /客户端直传 OSS/);
assert.match(settings, /directUpload:"0"/);
assert.match(settings, /ESA\/CNAME/);
assert.match(settings, /fileUpload\.aliyun\.domain/);
assert.match(settings, /旧版客户端的服务端上传也统一走 ESA/);
assert.match(settings, /"http-request":t\.imgoUploadImageRequest/);
assert.match(settings, /window\.imgoAdminElementUpload/);
assert.match(app, /window\.imgoAdminElementUpload/);

console.log('direct upload checks passed');
