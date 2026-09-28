const assert = require('assert');
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const { execFileSync } = require('child_process');

const root = path.resolve(__dirname, '..');
const read = relative => fs.readFileSync(path.join(root, relative), 'utf8');
const component = { methods: {} };
vm.runInNewContext(read('frontend/contact-search.js'), { component, Object, Array, Number, String });

const contacts = [
  { id: 2, account: 'wuhu2', displayName: '小王', realname: '王小明', name_py: 'wangxiaoming', is_group: 0 },
  { id: 3, account: 'alpha', displayName: null, realname: null, name_py: null, is_group: 0 },
  { id: 'group-1', displayName: '测试群', name_py: 'ceshiqun', is_group: 1 }
];
const instance = { keywords: '', searchList: [] };
const search = query => {
  instance.keywords = query;
  component.methods.searchContact.call(instance, contacts);
  return instance.searchList;
};

assert.deepStrictEqual(Array.from(search('WUHU2'), item => item.id), [2], '账号搜索应忽略大小写');
assert.deepStrictEqual(Array.from(search('小王'), item => item.id), [2], '应支持昵称或备注搜索');
assert.deepStrictEqual(Array.from(search('王小明'), item => item.id), [2], '应支持真实姓名搜索');
assert.deepStrictEqual(Array.from(search('wang'), item => item.id), [2], '应支持拼音搜索');
assert.deepStrictEqual(Array.from(search('测试'), item => item.id), ['group-1'], '应支持群名搜索');
assert.strictEqual(search('wuhu2')[0].displayName, 'wuhu2（小王）', '搜索结果应显示账号和昵称');
assert.doesNotThrow(() => search('missing'), '空字段不应导致搜索异常');
assert.strictEqual(search('   ').length, 0, '空搜索应清空结果');

execFileSync(process.execPath, ['scripts/build-maintenance.cjs'], { cwd: root, stdio: 'inherit' });
const app = read('public/assets/js/app.85372e4e.js');
assert.ok(app.includes('IMGO_CONTACT_SEARCH_BEGIN'), '联系人搜索适配器必须打包');
assert.ok(app.includes('placeholder:"\u641c\u7d22\u8d26\u53f7\u3001\u6635\u79f0\u6216\u7fa4聊"'), '搜索框应说明支持的字段');
assert.ok(app.includes('autocomplete:"one-time-code",name:"imgo-contact-search"'), '搜索框应防止浏览器误自动填充');
assert.ok(app.includes('t.keywords&&0==t.searchList.length'), '只有输入关键字后才显示无结果提示');

console.log('Contact search tests passed');
