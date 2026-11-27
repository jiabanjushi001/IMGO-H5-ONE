const ImgoAutoGroupPicker = {
  name: 'ImgoAutoGroupPicker',
  props: {
    value: { type: Object, required: true }
  },
  data() {
    return { groups: [], loading: false, error: '', requestSerial: 0 }
  },
  computed: {
    mode() { return this.value && this.value.mode === 'existing' ? 'existing' : 'create' },
    groupID() { return Number((this.value || {}).group_id || 0) }
  },
  mounted() {
    if (!this.value.mode || this.value.group_id === undefined) {
      this.update({ mode: this.mode, group_id: this.groupID })
    }
    this.loadGroups()
  },
  beforeDestroy() { this.requestSerial++ },
  methods: {
    update(patch) {
      this.$emit('input', Object.assign({}, this.value || {}, patch))
    },
    setMode(mode) {
      this.update({ mode: mode === 'existing' ? 'existing' : 'create' })
      if (mode === 'existing' && !this.groups.length) this.loadGroups()
    },
    setGroup(groupID) { this.update({ group_id: Number(groupID || 0) }) },
    async loadGroups() {
      if (this.loading) return
      const serial = ++this.requestSerial
      this.loading = true
      this.error = ''
      const groups = []
      try {
        let page = 1
        while (page <= 50) {
          const response = await this.$api.groupApi.getGroupList({ page, limit: 200 })
          if (Number(response.code) !== 0) throw Error(response.msg || '读取群聊失败')
          if (!Array.isArray(response.data)) throw Error('群聊数据无效')
          groups.push(...response.data)
          if (response.data.length < 200 || groups.length >= Number(response.count || 0)) break
          page++
        }
        if (serial === this.requestSerial) {
          this.groups = groups.filter((item, index, all) => Number(item.group_id) > 0 && all.findIndex(candidate => Number(candidate.group_id) === Number(item.group_id)) === index)
        }
      } catch (error) {
        if (serial === this.requestSerial) this.error = error.message || '读取群聊失败'
      } finally {
        if (serial === this.requestSerial) this.loading = false
      }
    }
  },
  render(h) {
    const mode = h('el-radio-group', {
      props: { value: this.mode, size: 'small' },
      on: { input: this.setMode }
    }, [
      h('el-radio-button', { props: { label: 'create' } }, '自动新建群'),
      h('el-radio-button', { props: { label: 'existing' } }, '加入已有群')
    ])
    const children = [
      h('div', { class: 'lz-flex mt-10 imgo-auto-group-mode' }, [
        h('span', { class: 'c-999 f-12 imgo-auto-group-label' }, '加入方式：'), mode
      ])
    ]
    if (this.mode === 'existing') {
      children.push(h('div', { class: 'lz-flex mt-10 imgo-auto-group-existing' }, [
        h('span', { class: 'c-999 f-12 imgo-auto-group-label' }, '选择群聊：'),
        h('el-select', {
          props: { value: this.groupID || '', filterable: true, clearable: true, loading: this.loading, placeholder: '请选择已经创建好的群聊' },
          class: 'imgo-auto-group-select',
          style: { width: '360px', maxWidth: 'calc(100vw - 190px)' },
          on: { input: this.setGroup }
        }, this.groups.map(group => h('el-option', {
          key: Number(group.group_id),
          props: { value: Number(group.group_id), label: `${group.name || '未命名群聊'}（群号 ${group.group_id}，${Number(group.user_count || 0)}人）` }
        }))),
        h('el-button', { props: { type: 'text', loading: this.loading }, on: { click: this.loadGroups } }, '刷新群列表')
      ]))
      if (this.error) children.push(h('div', { class: 'c-danger f-12 mt-5' }, this.error))
    }
    return h('div', { class: 'imgo-auto-group-picker' }, children)
  }
}
