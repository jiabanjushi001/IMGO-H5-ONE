// Presentational SVG chart. Null values are gaps, never fabricated zeros.
const ImgoOverviewChart = {
  props: { title: String, labels: Array, series: Array, bars: Boolean, note: String },
  render(h) {
    const labels = this.labels || [], series = this.series || []
    const max = Math.max(1, ...series.flatMap(s => s.values.filter(v => v != null)))
    const top = Math.max(1, Math.ceil(max / 4)) * 4
    const x = i => 48 + i * 680 / Math.max(1, labels.length - 1)
    const y = v => 225 - Number(v) / top * 185
    const marks = []
    for (let i = 0; i <= 4; i++) {
      const v = top * i / 4
      marks.push(h('line', { attrs: { x1: 48, x2: 728, y1: y(v), y2: y(v), stroke: '#e5ebf3' } }), h('text', { attrs: { x: 37, y: y(v) + 4, 'text-anchor': 'end' } }, String(v)))
    }
    labels.forEach((label, i) => {
      if (i % Math.max(1, Math.ceil(labels.length / 6)) === 0 || i === labels.length - 1) marks.push(h('text', { attrs: { x: x(i), y: 249, 'text-anchor': 'middle' } }, label))
    })
    series.forEach((s, si) => {
      if (this.bars) {
        const w = Math.min(18, 600 / Math.max(1, labels.length) / series.length)
        s.values.forEach((v, i) => { if (v != null) marks.push(h('rect', { attrs: { x: x(i) + (si - series.length / 2) * w, y: y(v), width: w - 2, height: 225 - y(v), rx: 3, fill: s.color } }, [h('title', `${labels[i]} ${s.name}：${v}`)])) })
      } else {
        let path = '', connected = false
        s.values.forEach((v, i) => { if (v == null) { connected = false; return }; path += `${connected ? 'L' : 'M'}${x(i)},${y(v)} `; connected = true })
        marks.push(h('path', { attrs: { d: path, fill: 'none', stroke: s.color, 'stroke-width': 2 } }))
        s.values.forEach((v, i) => { if (v != null) marks.push(h('circle', { attrs: { cx: x(i), cy: y(v), r: 3, fill: s.color } }, [h('title', `${labels[i]} ${s.name}：${v}`)])) })
      }
    })
    return h('section', { class: 'imgo-chart' }, [
      h('header', [h('span', this.title), h('div', this.$slots.controls)]),
      h('div', { class: 'imgo-chart-legend' }, series.map(s => h('span', [h('i', { style: { background: s.color } }), s.name]))),
      h('svg', { attrs: { viewBox: '0 0 770 275', role: 'img', 'aria-label': this.title } }, [h('title', this.title), ...marks]),
      h('p', { class: 'imgo-chart-note' }, this.note || '统计时区：北京时间；鼠标移至数据点查看数值。')
    ])
  }
}
