/* IMGO_CHAT_ACCOUNT_SWITCHER_BEGIN */
var imgoAccountVaultStorageKey = 'imgo:chat-accounts:v1';
var imgoAccountVaultLimit = 50;

function imgoAccountVaultCookie() {
  return o();
}

function imgoAccountVaultParseUserInfo(value) {
  if (!value) return {};
  if (typeof value === 'object') return value;
  try {
    return JSON.parse(value);
  } catch (error) {
    return {};
  }
}

function imgoAccountVaultNormalize(input) {
  input = input || {};
  var userInfo = imgoAccountVaultParseUserInfo(input.userInfo || input.user_info);
  var safeUserInfo = Object.assign({}, userInfo);
  delete safeUserInfo.password;
  delete safeUserInfo.password_confirmation;
  delete safeUserInfo.pay_password;
  delete safeUserInfo.google_code;
  var userId = userInfo.user_id != null ? userInfo.user_id : input.user_id;
  var account = userInfo.account || input.account || '';
  var authToken = input.authToken || input.auth_token || '';
  var sessionId = input.sessionId || input.session_id || '';
  if ((!userId && !account) || !authToken || !sessionId) return null;
  return {
    key: userId != null && String(userId) !== '' ? 'id:' + String(userId) : 'account:' + String(account),
    user_id: userId,
    account: String(account || ''),
    realname: String(userInfo.realname || input.realname || account || ('账号 ' + userId)),
    avatar: String(userInfo.avatar || input.avatar || ''),
    authToken: String(authToken),
    sessionId: String(sessionId),
    userInfo: Object.assign(safeUserInfo, {
      user_id: userId,
      account: account,
      realname: userInfo.realname || input.realname || account || ('账号 ' + userId),
      avatar: userInfo.avatar || input.avatar || ''
    }),
    updatedAt: Number(input.updatedAt) || Date.now()
  };
}

var imgoAccountVault = {
  read: function () {
    try {
      var value = JSON.parse(window.localStorage.getItem(imgoAccountVaultStorageKey) || '[]');
      if (!Array.isArray(value)) return [];
      return value.map(imgoAccountVaultNormalize).filter(Boolean).slice(0, imgoAccountVaultLimit);
    } catch (error) {
      return [];
    }
  },
  write: function (accounts) {
    var safeAccounts = (Array.isArray(accounts) ? accounts : [])
      .map(imgoAccountVaultNormalize)
      .filter(Boolean)
      .sort(function (left, right) { return right.updatedAt - left.updatedAt; })
      .slice(0, imgoAccountVaultLimit);
    try {
      window.localStorage.setItem(imgoAccountVaultStorageKey, JSON.stringify(safeAccounts));
    } catch (error) {
      return [];
    }
    return safeAccounts;
  },
  save: function (input) {
    var account = imgoAccountVaultNormalize(input);
    if (!account) return this.read();
    var accounts = this.read().filter(function (item) {
      return item.key !== account.key && item.authToken !== account.authToken;
    });
    accounts.unshift(account);
    return this.write(accounts);
  },
  captureCurrent: function (userInfo) {
    var cookies = imgoAccountVaultCookie();
    return this.save({
      authToken: cookies.get('authToken'),
      sessionId: cookies.get('sessionId'),
      userInfo: userInfo || cookies.get('UserInfo')
    });
  },
  currentToken: function () {
    return String(imgoAccountVaultCookie().get('authToken') || '');
  },
  switchTo: function (account) {
    var normalized = imgoAccountVaultNormalize(account);
    if (!normalized) throw new Error('账号凭证不完整，请重新登录该账号');
    normalized.updatedAt = Date.now();
    this.save(normalized);
    var cookies = imgoAccountVaultCookie();
    cookies.set('authToken', normalized.authToken);
    cookies.set('sessionId', normalized.sessionId);
    cookies.set('UserInfo', normalized.userInfo);
    return normalized;
  },
  remove: function (key) {
    return this.write(this.read().filter(function (item) { return item.key !== key; }));
  },
  removeCurrent: function () {
    var token = this.currentToken();
    return this.write(this.read().filter(function (item) { return item.authToken !== token; }));
  },
  clearActive: function () {
    var cookies = imgoAccountVaultCookie();
    cookies.rm('authToken');
    cookies.rm('sessionId');
    cookies.rm('UserInfo');
  }
};

window.ImgoAccountVault = imgoAccountVault;

var ImgoAccountSwitcher = {
  name: 'ImgoAccountSwitcher',
  data: function () {
    return {
      visible: false,
      accounts: []
    };
  },
  methods: {
    refresh: function () {
      this.accounts = imgoAccountVault.read();
    },
    open: function () {
      imgoAccountVault.captureCurrent((this.$store.state || {}).userInfo);
      this.refresh();
      this.visible = true;
    },
    isCurrent: function (account) {
      return account.authToken === imgoAccountVault.currentToken();
    },
    accountSubtitle: function (account) {
      var parts = [];
      if (account.account) parts.push('账号：' + account.account);
      if (account.user_id != null && String(account.user_id) !== '') parts.push('ID：' + account.user_id);
      return parts.join(' · ');
    },
    switchAccount: function (account) {
      if (this.isCurrent(account)) {
        this.visible = false;
        return;
      }
      try {
        imgoAccountVault.switchTo(account);
        this.$message.success('正在切换到 ' + account.realname);
        window.setTimeout(function () { window.location.reload(); }, 120);
      } catch (error) {
        this.$message.error(error.message || '切换失败，请重新登录');
      }
    },
    removeAccount: function (account) {
      var self = this;
      if (this.isCurrent(account)) return;
      this.$confirm('仅从这台设备移除“' + account.realname + '”，不会退出其他账号。', '移除账号', {
        confirmButtonText: '移除',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(function () {
        imgoAccountVault.remove(account.key);
        self.refresh();
      }).catch(function () {});
    },
    addAccount: function () {
      imgoAccountVault.captureCurrent((this.$store.state || {}).userInfo);
      imgoAccountVault.clearActive();
      var base = window.location.origin + window.location.pathname;
      window.location.replace(base + '#/login?addAccount=1');
      window.location.reload();
    },
    avatarText: function (account) {
      return String(account.realname || account.account || '?').slice(0, 1);
    },
    renderAccount: function (h, account) {
      var self = this;
      var current = this.isCurrent(account);
      return h('div', {
        key: account.key,
        class: ['imgo-account-switcher__item', current ? 'is-current' : '']
      }, [
        h('el-avatar', { props: { size: 44, src: account.avatar } }, [this.avatarText(account)]),
        h('div', { class: 'imgo-account-switcher__identity' }, [
          h('div', { class: 'imgo-account-switcher__name' }, [
            h('span', [account.realname]),
            current ? h('span', { class: 'imgo-account-switcher__current' }, ['当前账号']) : null
          ]),
          h('div', { class: 'imgo-account-switcher__account' }, [this.accountSubtitle(account)])
        ]),
        h('div', { class: 'imgo-account-switcher__actions' }, [
          h('el-button', {
            props: { type: current ? 'info' : 'primary', size: 'mini', disabled: current },
            on: { click: function () { self.switchAccount(account); } }
          }, [current ? '使用中' : '切换']),
          !current ? h('el-button', {
            props: { type: 'text', size: 'mini' },
            class: 'imgo-account-switcher__remove',
            attrs: { 'aria-label': '移除 ' + account.realname },
            on: { click: function () { self.removeAccount(account); } }
          }, ['移除']) : null
        ])
      ]);
    }
  },
  render: function (h) {
    var self = this;
    var body = this.accounts.length
      ? h('div', { class: 'imgo-account-switcher__list' }, this.accounts.map(function (account) {
          return self.renderAccount(h, account);
        }))
      : h('div', { class: 'imgo-account-switcher__empty' }, [
          h('i', { class: 'el-icon-user' }),
          h('p', ['还没有保存的账号'])
        ]);
    return h('el-dialog', {
      class: 'imgo-account-switcher-shell',
      props: {
        visible: this.visible,
        title: '切换账号',
        width: '480px',
        appendToBody: true,
        closeOnClickModal: false,
        customClass: 'imgo-account-switch-dialog'
      },
      on: {
        'update:visible': function (value) { self.visible = value; },
        close: function () { self.visible = false; }
      }
    }, [
      h('div', { class: 'imgo-account-switcher__tip' }, ['账号凭证仅保存在当前设备，切换后会自动重新连接聊天。']),
      body,
      h('div', { slot: 'footer', class: 'imgo-account-switcher__footer' }, [
        h('el-button', { props: { type: 'primary' }, on: { click: this.addAccount } }, [
          h('i', { class: 'el-icon-plus' }),
          ' 登录其他账号'
        ])
      ])
    ]);
  }
};

component.components = Object.assign({}, component.components || {}, {
  ImgoAccountSwitcher: ImgoAccountSwitcher
});

var imgoAccountOriginalMounted = component.mounted;
component.mounted = function () {
  if (Array.isArray(imgoAccountOriginalMounted)) {
    imgoAccountOriginalMounted.forEach(function (hook) {
      if (typeof hook === 'function') hook.call(this);
    }, this);
  } else if (typeof imgoAccountOriginalMounted === 'function') {
    imgoAccountOriginalMounted.call(this);
  }
  imgoAccountVault.captureCurrent((this.$store.state || {}).userInfo);
};
/* IMGO_CHAT_ACCOUNT_SWITCHER_END */
