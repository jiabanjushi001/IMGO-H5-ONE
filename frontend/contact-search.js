/* IMGO_CONTACT_SEARCH_BEGIN */
var imgoContactSearchOriginalMethods = component.methods || {};

function imgoContactSearchText(value) {
  return value == null ? '' : String(value).toLocaleLowerCase();
}

function imgoContactSearchLabel(contact) {
  if (!contact || Number(contact.is_group) !== 0) return contact && contact.displayName;
  var account = contact.account == null ? '' : String(contact.account).trim();
  var displayName = contact.displayName == null ? '' : String(contact.displayName).trim();
  if (account && displayName && account !== displayName) return account + '（' + displayName + '）';
  return account || displayName;
}

component.methods = Object.assign({}, imgoContactSearchOriginalMethods, {
  searchContact: function (contacts) {
    var query = imgoContactSearchText(this.keywords).trim();
    if (!query) {
      this.searchList = [];
      return;
    }
    var source = Array.isArray(contacts) ? contacts : [];
    this.searchList = source.filter(function (contact) {
      return ['account', 'displayName', 'realname', 'name_py'].some(function (field) {
        return imgoContactSearchText(contact && contact[field]).indexOf(query) !== -1;
      });
    }).map(function (contact) {
      var result = Object.assign({}, contact);
      result.displayName = imgoContactSearchLabel(contact);
      return result;
    });
  }
});
/* IMGO_CONTACT_SEARCH_END */
