<template>
  <div class="seedance-config">
    <div class="config-section">
      <h3>Seedance 视频生成配置</h3>
      <el-alert type="info" :closable="false" style="margin-bottom: 16px">
        配置 Seedance 2.0 视频生成 API 连接信息。模型 ID 可在 Seedance 控制台获取。
      </el-alert>

      <el-form :model="config" label-width="120px">
        <el-form-item label="API URL">
          <el-input v-model="config.api_url" placeholder="http://118.196.64.1/api/v1" />
        </el-form-item>
        <el-form-item label="Bearer Token">
          <el-input v-model="config.bearer_token" type="password" show-password placeholder="输入 API Token" />
        </el-form-item>
        <el-form-item label="快速模型 ID">
          <el-input v-model="config.model_fast" placeholder="ep-20260307130821-xw5wf" />
        </el-form-item>
        <el-form-item label="标准模型 ID">
          <el-input v-model="config.model_std" placeholder="ep-20260307130721-bx7tv" />
        </el-form-item>
      </el-form>
    </div>

    <div class="config-section">
      <h3>算力配置</h3>
      <el-form :model="config.power" label-width="140px">
        <el-form-item label="文生视频">
          <el-input-number v-model="config.power.text_to_video" :min="0" />
        </el-form-item>
        <el-form-item label="图生视频-首帧">
          <el-input-number v-model="config.power.image_to_video_first" :min="0" />
        </el-form-item>
        <el-form-item label="图生视频-首尾帧">
          <el-input-number v-model="config.power.image_to_video_dual" :min="0" />
        </el-form-item>
        <el-form-item label="多模态参考">
          <el-input-number v-model="config.power.multimodal_ref" :min="0" />
        </el-form-item>
        <el-form-item label="编辑视频">
          <el-input-number v-model="config.power.edit_video" :min="0" />
        </el-form-item>
        <el-form-item label="延长视频">
          <el-input-number v-model="config.power.extend_video" :min="0" />
        </el-form-item>
        <el-form-item label="虚拟人像">
          <el-input-number v-model="config.power.virtual_avatar" :min="0" />
        </el-form-item>
      </el-form>
    </div>

    <div class="btn-area">
      <el-button type="primary" @click="save">保存</el-button>
      <el-button @click="load">重置</el-button>
    </div>
  </div>
</template>

<script setup>
import { httpGet, httpPost } from '@/utils/http'
import { showMessageOK, showMessageError } from '@/utils/dialog'
import { onMounted, reactive } from 'vue'

const config = reactive({
  api_url: 'http://118.196.64.1/api/v1',
  bearer_token: '',
  model_fast: 'ep-20260307130821-xw5wf',
  model_std: 'ep-20260307130721-bx7tv',
  power: {
    text_to_video: 30,
    image_to_video_first: 35,
    image_to_video_dual: 40,
    multimodal_ref: 50,
    edit_video: 45,
    extend_video: 50,
    virtual_avatar: 35,
  },
})

const load = async () => {
  try {
    const res = await httpGet('/api/admin/config/get?key=seedance')
    if (res.data) {
      Object.assign(config, res.data)
      if (res.data.power) Object.assign(config.power, res.data.power)
    }
  } catch (e) {
    // 配置不存在，使用默认值
  }
}

const save = async () => {
  try {
    await httpPost('/api/admin/seedance/config/update', config)
    showMessageOK('保存成功')
  } catch (e) {
    showMessageError('保存失败: ' + e.message)
  }
}

onMounted(load)
</script>

<style scoped>
.seedance-config { padding: 20px; }
.config-section { background: #fff; border-radius: 8px; padding: 20px; margin-bottom: 16px; }
.config-section h3 { margin: 0 0 16px; font-size: 16px; }
.btn-area { padding: 0 20px; }
</style>
