// Vue 2 component embedded in the existing compiled member page.
// The parent owns query state; this toolbar only emits filter changes and searches.
const ImgoMemberReferralFilter = {
  name: 'ImgoMemberReferralFilter',
  props: {
    scope: { type: String, default: '' },
    referrerAccount: { type: String, default: '' }
  },
  render(h) {
    return h('div', { class: 'imgo-member-referral-filter' }, [
      h('span', { class: 'imgo-member-referral-label' }, '层级'),
      h('el-select', {
        props: { value: this.scope, clearable: true, placeholder: '不选则查用户本人' },
        on: {
          input: value => this.$emit('update:scope', value),
          change: () => this.$emit('search')
        }
      }, [
        h('el-option', { props: { label: '直属下级', value: 'direct' } }),
        h('el-option', { props: { label: '全部下级', value: 'all' } })
      ]),
      h('el-button', {
        props: { type: 'primary', icon: 'el-icon-search' },
        on: { click: () => this.$emit('search') }
      }, '查询')
    ])
  }
};
