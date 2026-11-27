const assert = require('assert');
const fs = require('fs');
const path = require('path');
const vm = require('vm');

const root = path.resolve(__dirname, '..');
const adapter = fs.readFileSync(path.join(root, 'frontend/group-member-search.js'), 'utf8');
let loadedGroupId = null;
const component = {
  data() { return { groupUser: [] }; },
  computed: {},
  methods: {
    getGroupUserList(groupId) {
      loadedGroupId = groupId;
      return 'loaded';
    }
  }
};

vm.runInNewContext(adapter, { component, Object, Array, String });

const state = component.data();
state.groupUser = [
  { userInfo: { account: 'wuhu001', displayName: '张三' } },
  { userInfo: { account: 'TEST002', nickname: '小李' } },
  { userInfo: { account: 'user003', realname: '王老师' } }
];

state.imgoGroupMemberKeyword = '';
assert.strictEqual(component.computed.imgoFilteredGroupUsers.call(state).length, 3);
state.imgoGroupMemberKeyword = '张三';
assert.deepStrictEqual(Array.from(component.computed.imgoFilteredGroupUsers.call(state), item => item.userInfo.account), ['wuhu001']);
state.imgoGroupMemberKeyword = 'test';
assert.deepStrictEqual(Array.from(component.computed.imgoFilteredGroupUsers.call(state), item => item.userInfo.account), ['TEST002']);
state.imgoGroupMemberKeyword = '老师';
assert.deepStrictEqual(Array.from(component.computed.imgoFilteredGroupUsers.call(state), item => item.userInfo.account), ['user003']);
state.imgoGroupMemberKeyword = '不存在';
assert.strictEqual(component.computed.imgoFilteredGroupUsers.call(state).length, 0);

state.imgoGroupMemberKeyword = '保留';
state.imgoGroupMemberSearchGroupId = '17';
assert.strictEqual(component.methods.getGroupUserList.call(state, 17), 'loaded');
assert.strictEqual(state.imgoGroupMemberKeyword, '保留');
assert.strictEqual(loadedGroupId, 17);
component.methods.getGroupUserList.call(state, 18);
assert.strictEqual(state.imgoGroupMemberKeyword, '');

const bundle = fs.readFileSync(path.join(root, 'public/assets/js/app.85372e4e.js'), 'utf8');
assert(bundle.includes('IMGO_GROUP_MEMBER_SEARCH_BEGIN'));
assert(bundle.includes('搜索群成员昵称或账号'));
assert(bundle.includes('t._l(t.imgoFilteredGroupUsers'));
assert(bundle.includes('未找到群成员'));

const css = fs.readFileSync(path.join(root, 'frontend/chat-design.css'), 'utf8');
assert(css.includes('.imgo-group-member-search'));
console.log('chat group-member search tests passed');
