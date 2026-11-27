function imgoCanUsePreparedVideo(userInfo) {
  userInfo = userInfo || {};
  var userId = Number(userInfo.user_id || userInfo.id || 0);
  return userId === 1 || Number(userInfo.agent_mode || 0) === 1;
}

component.components = Object.assign({}, component.components, {
  ImgoPreparedVideoDialog: {
    name: 'ImgoPreparedVideoDialog',
    props: {
      visible: { type: Boolean, default: false }
    },
    data: function () {
      return { loading: false, uploading: false, videos: [] };
    },
    watch: {
      visible: function (value) {
        if (value) this.load();
      }
    },
    methods: {
      close: function () {
        this.$emit('update:visible', false);
      },
      load: async function () {
        if (this.loading) return;
        this.loading = true;
        try {
          var result = await this.$api.imApi.getCallVideosAPI({ page: 1, limit: 100 });
          this.videos = Number(result && result.code) === 0 && Array.isArray(result.data) ? result.data : [];
        } catch (error) {
          this.videos = [];
          this.$message.error('预设视频加载失败');
        } finally {
          this.loading = false;
        }
      },
      chooseLive: function () {
        this.$emit('select', { mode: 'live', video: null });
      },
      choosePrepared: function (video) {
        this.$emit('select', { mode: 'recorded', video: video });
      },
      openUpload: function () {
        if (!this.uploading && this.$refs.videoFile) this.$refs.videoFile.click();
      },
      upload: async function (event) {
        var input = event && event.target;
        var file = input && input.files && input.files[0];
        if (!file || this.uploading) return;
        if (file.type && file.type.indexOf('video/') !== 0) {
          this.$message.error('请选择视频文件');
          input.value = '';
          return;
        }
        this.uploading = true;
        var loading = this.$loading({ lock: true, text: '视频上传中…' });
        try {
          var form = new FormData();
          form.append('file', file);
          var extension = String(file.name || '').split('.').pop();
          if (extension) form.append('ext', extension);
          var result = await this.$api.imApi.sendFileAPI(form);
          if (Number(result && result.code) !== 0 || !Number(result && result.data && result.data.file_id)) {
            throw new Error(result && result.msg || '视频上传失败');
          }
          await this.load();
          this.$message.success('预设视频上传成功');
        } catch (error) {
          this.$message.error(error && error.message || '视频上传失败');
        } finally {
          loading.close();
          this.uploading = false;
          if (input) input.value = '';
        }
      },
      formatSize: function (value) {
        var size = Number(value || 0);
        if (size < 1024 * 1024) return Math.max(1, Math.round(size / 1024)) + ' KB';
        return (size / 1024 / 1024).toFixed(1) + ' MB';
      }
    },
    render: function (h) {
      var self = this;
      var list = this.loading
        ? h('div', { class: 'imgo-prepared-video-empty' }, ['正在加载…'])
        : this.videos.length
          ? h('div', { class: 'imgo-prepared-video-list' }, this.videos.map(function (video) {
              return h('button', {
                key: video.file_id,
                class: 'imgo-prepared-video-row',
                attrs: { type: 'button' },
                on: { click: function () { self.choosePrepared(video); } }
              }, [
                h('i', { class: 'el-icon-video-play' }),
                h('span', { class: 'imgo-prepared-video-name' }, [video.name || '未命名视频']),
                h('small', [self.formatSize(video.size)]),
                h('strong', ['选择'])
              ]);
            }))
          : h('div', { class: 'imgo-prepared-video-empty' }, ['还没有预设视频，请先上传']);
      return h('el-dialog', {
        class: 'imgo-prepared-video-shell',
        props: {
          visible: this.visible,
          title: '选择视频通话方式',
          width: '520px',
          appendToBody: true,
          closeOnClickModal: false
        },
        on: {
          'update:visible': function (value) { self.$emit('update:visible', value); },
          close: this.close
        }
      }, [
        h('button', {
          class: 'imgo-prepared-video-live',
          attrs: { type: 'button' },
          on: { click: this.chooseLive }
        }, [
          h('i', { class: 'el-icon-video-camera' }),
          h('span', [h('strong', ['普通视频']), h('small', ['使用实时摄像头'])]),
          h('b', ['开始'])
        ]),
        h('div', { class: 'imgo-prepared-video-heading' }, [
          h('div', [h('strong', ['预设视频']), h('small', ['选择已上传的视频进行通话'])]),
          h('el-button', {
            props: { type: 'primary', size: 'small', loading: this.uploading },
            on: { click: this.openUpload }
          }, [this.uploading ? '上传中' : '上传视频']),
          h('input', {
            ref: 'videoFile',
            attrs: { type: 'file', accept: 'video/*' },
            style: { display: 'none' },
            on: { change: this.upload }
          })
        ]),
        list
      ]);
    }
  }
});

var imgoPreparedVideoBaseData = component.data;
component.data = function () {
  return Object.assign(imgoPreparedVideoBaseData.call(this), {
    preparedVideoDialog: false,
    preparedVideoMode: '',
    preparedVideoFileId: 0,
    preparedVideoUrl: ''
  });
};

var imgoPreparedVideoBaseCalled = component.methods.called;
component.methods.called = function (type) {
  if (Number(type) === 1 && imgoCanUsePreparedVideo(this.userInfo)) {
    this.preparedVideoDialog = true;
    return;
  }
  this.preparedVideoMode = Number(type) === 1 ? 'live' : '';
  this.preparedVideoFileId = 0;
  this.preparedVideoUrl = '';
  return imgoPreparedVideoBaseCalled.call(this, type);
};

component.methods.imgoStartVideoMode = function (selection) {
  selection = selection || {};
  var video = selection.video || {};
  if (selection.mode === 'recorded' && (!Number(video.file_id) || !video.url)) {
    this.$message.error('请选择有效的预设视频');
    return;
  }
  this.preparedVideoMode = selection.mode === 'recorded' ? 'recorded' : 'live';
  this.preparedVideoFileId = this.preparedVideoMode === 'recorded' ? Number(video.file_id) : 0;
  this.preparedVideoUrl = this.preparedVideoMode === 'recorded' ? String(video.url) : '';
  this.preparedVideoDialog = false;
  this.$nextTick(function () {
    imgoPreparedVideoBaseCalled.call(this, 1);
  }.bind(this));
};
