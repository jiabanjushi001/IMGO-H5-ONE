/* IMGO_MOBILE_CHAT_BEGIN */
var imgoMobileOriginalData = component.data;
var imgoMobileOriginalMethods = component.methods || {};
var imgoMobileOriginalOpenChat = imgoMobileOriginalMethods.openChat;
var imgoMobileOriginalChangeContact = imgoMobileOriginalMethods.handleChangeContact;
var imgoMobileOriginalMounted = component.mounted;
var imgoMobileOriginalBeforeDestroy = component.beforeDestroy;

function imgoMobileCallHook(hook, instance) {
  if (Array.isArray(hook)) {
    hook.forEach(function (item) {
      if (typeof item === 'function') item.call(instance);
    });
    return;
  }
  if (typeof hook === 'function') hook.call(instance);
}

component.data = function () {
  var state = typeof imgoMobileOriginalData === 'function'
    ? imgoMobileOriginalData.apply(this, arguments)
    : {};
  state = state || {};
  state.imgoMobileConversation = false;
  return state;
};

component.methods = Object.assign({}, imgoMobileOriginalMethods, {
  openChat: function (contactId, imui) {
    this.keywords = '';
    if (imui && typeof imui.changeContact === 'function') {
      return imui.changeContact(contactId, 'messages');
    }
    if (typeof imgoMobileOriginalOpenChat === 'function') {
      return imgoMobileOriginalOpenChat.apply(this, arguments);
    }
  },
  handleChangeContact: function () {
    var result;
    var imui = arguments[1];
    if (typeof window !== 'undefined' && window.matchMedia && window.matchMedia('(max-width: 700px)').matches) {
      this.imgoMobileConversation = true;
    }
    if (imui && imui.activeSidebar === 'contacts' && imui.currentContactIdSidebarContact != null && typeof imui.changeContact === 'function') {
      return imui.changeContact(imui.currentContactIdSidebarContact, 'messages');
    }
    if (typeof imgoMobileOriginalChangeContact === 'function') {
      result = imgoMobileOriginalChangeContact.apply(this, arguments);
    }
    return result;
  },
  imgoMobileBack: function () {
    this.imgoMobileConversation = false;
  }
});

component.mounted = function () {
  imgoMobileCallHook(imgoMobileOriginalMounted, this);
  if (typeof window === 'undefined' || !window.matchMedia) return;
  this.imgoMobileMediaQuery = window.matchMedia('(max-width: 700px)');
  this.imgoMobileMediaListener = function (event) {
    if (!event.matches) this.imgoMobileConversation = false;
  }.bind(this);
  if (this.imgoMobileMediaQuery.addEventListener) {
    this.imgoMobileMediaQuery.addEventListener('change', this.imgoMobileMediaListener);
  } else if (this.imgoMobileMediaQuery.addListener) {
    this.imgoMobileMediaQuery.addListener(this.imgoMobileMediaListener);
  }
};

component.beforeDestroy = function () {
  if (this.imgoMobileMediaQuery && this.imgoMobileMediaListener) {
    if (this.imgoMobileMediaQuery.removeEventListener) {
      this.imgoMobileMediaQuery.removeEventListener('change', this.imgoMobileMediaListener);
    } else if (this.imgoMobileMediaQuery.removeListener) {
      this.imgoMobileMediaQuery.removeListener(this.imgoMobileMediaListener);
    }
  }
  imgoMobileCallHook(imgoMobileOriginalBeforeDestroy, this);
};
/* IMGO_MOBILE_CHAT_END */
