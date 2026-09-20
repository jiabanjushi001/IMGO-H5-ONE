// The original UniApp compiler is unavailable in this checkout. Keep the
// existing H5 preview synchronized with pages/message/chat.vue until a full build.
import { createHash } from 'node:crypto'
import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const root = resolve(import.meta.dirname, '../unpackage/dist/build/h5')
const read = path => readFileSync(resolve(root, path), 'utf8')
const write = (path, value) => writeFileSync(resolve(root, path), value)
const hash = value => createHash('sha256').update(value).digest('hex').slice(0, 12)

let html = read('index.html')
const entry = html.match(/src="\/assets\/(index-[^"/]+\.js)"/)
if (!entry) throw Error('Active H5 entry was not found')
let main = read(`assets/${entry[1]}`)
const chat = main.match(/pages-message-chat\.[A-Za-z0-9_-]+\.js/)
if (!chat) throw Error('Active group chat chunk was not found')
let page = read(`assets/${chat[0]}`)

const requestBefore = 'getMessageList(){let t={is_group:this.is_group,toContactId:this.contact.id,page:this.page,limit:this.limit};'
const requestAfter = 'getMessageList(){let t={is_group:this.is_group,toContactId:this.contact.id,page:this.page,limit:this.limit,type:1==this.is_group?"all":""};'
if (page.includes(requestBefore)) page = page.replace(requestBefore, requestAfter)
else if (!page.includes(requestAfter)) throw Error('Group message request changed; inspect compiled chunk')

const eventBefore = '["event"==e.type?(a(),o(r,{key:0,class:"cu-info"},'
const eventAfter = '["event"==e.type?(a(),o(r,{key:0,class:"cu-info",style:1==l.is_group?{display:"none"}:null},'
if (page.includes(eventBefore)) page = page.replace(eventBefore, eventAfter)
else if (!page.includes(eventAfter)) throw Error('Group event renderer changed; inspect compiled chunk')

const chatName = `pages-message-chat.imgo${hash(page)}.js`
write(`assets/${chatName}`, page)
main = main.replaceAll(chat[0], chatName)
const entryName = `index-imgo${hash(main)}.js`
write(`assets/${entryName}`, main)
html = html.replace(entry[1], entryName)
write('index.html', html)
console.log(`Updated active H5 preview: ${chatName}`)
