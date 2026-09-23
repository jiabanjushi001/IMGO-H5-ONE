const imgoAdminPermissionByPath = {
  '/manage/index': 'manage.overview',
  '/manage/setting': 'manage.settings',
  '/manage/user': 'manage.users',
  '/manage/message': 'manage.messages',
  '/manage/group': 'manage.groups',
  '/manage/files': 'manage.files',
  '/manage/bank': 'manage.bank',
  '/manage/finance/recharges': 'manage.finance',
  '/manage/finance/withdrawals': 'manage.finance'
}

function imgoBuildAdminMenu(userInfo, routes) {
  const user = userInfo || {}
  const isSuper = Number(user.user_id) === 1
  const permissions = new Set(Array.isArray(user.menu_permissions) ? user.menu_permissions : [])
  const normal = (routes || []).filter(route => {
    if (route.path === '/manage/wallet' || route.path.startsWith('/manage/finance/')) return false
    if (route.path === '/manage/role') return isSuper
    const permission = imgoAdminPermissionByPath[route.path]
    return isSuper || (!!permission && permissions.has(permission))
  })
  if (isSuper || permissions.has('manage.finance')) {
    const finance = {
      path: '/manage/finance', meta: { title: '财务', icon: 'el-icon-money' },
      children: [
        { path: '/manage/finance/recharges', meta: { title: '充值订单', icon: 'el-icon-document-add' } },
        { path: '/manage/finance/withdrawals', meta: { title: '提现订单', icon: 'el-icon-document-remove' } }
      ]
    }
    const bankIndex = normal.findIndex(route => route.path === '/manage/bank')
    normal.splice(bankIndex < 0 ? normal.length : bankIndex, 0, finance)
  }
  return normal
}

function imgoEnsureAuthorizedAdminRoute(vm) {
  const current = vm.$route.path
  if (!current.startsWith('/manage/')) return
  const allowed = []
  for (const route of vm.routes || []) {
    if (Array.isArray(route.children)) allowed.push(...route.children.map(child => child.path))
    else allowed.push(route.path)
  }
  if (allowed.includes(current) || (current === '/manage/wallet' && allowed.includes('/manage/finance/withdrawals'))) return
  const target = allowed[0] || '/chat'
  if (vm.$message) vm.$message.warning('无权操作')
  vm.$router.replace(target)
}
