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
        <el-form-item label="Seedance 2.0 模型 ID">
          <el-input v-model="config.model_std" placeholder="ep-20260307130721-bx7tv" />
        </el-form-item>
      </el-form>
    </div>

    <div class="config-section">
      <h3>算力配置（按模型 × 分辨率 × 时长秒数计费）</h3>
      <el-form label-width="220px">
        <el-divider content-position="left">Seedance 2.0 Fast 每秒算力</el-divider>
        <el-form-item label="480p">
          <el-input-number v-model="config.power.fast_price['480p']" :min="1" />
        </el-form-item>
        <el-form-item label="720p">
          <el-input-number v-model="config.power.fast_price['720p']" :min="1" />
        </el-form-item>
        <el-form-item label="1080p">
          <el-input-number v-model="config.power.fast_price['1080p']" :min="1" />
        </el-form-item>
        <el-divider content-position="left">Seedance 2.0 每秒算力</el-divider>
        <el-form-item label="480p">
          <el-input-number v-model="config.power.vip_price['480p']" :min="1" />
        </el-form-item>
        <el-form-item label="720p">
          <el-input-number v-model="config.power.vip_price['720p']" :min="1" />
        </el-form-item>
        <el-form-item label="1080p">
          <el-input-number v-model="config.power.vip_price['1080p']" :min="1" />
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

const defaultPower = {
  fast_price: { '480p': 3, '720p': 5, '1080p': 8 },
  vip_price: { '480p': 5, '720p': 8, '1080p': 12 },
}

const config = reactive({
  api_url: 'http://118.196.64.1/api/v1',
  bearer_token: '',
  model_fast: 'ep-20260307130821-xw5wf',
  model_std: 'ep-20260307130721-bx7tv',
  power: structuredClone(defaultPower),
})

function normalizePower(power = {}) {
  config.power.fast_price = { ...defaultPower.fast_price, ...(power.fast_price || {}) }
  config.power.vip_price = { ...defaultPower.vip_price, ...(power.vip_price || {}) }
}

const load = async () => {
  try {
    const res = await httpGet('/api/admin/config/get?key=seedance')
    if (res.data) {
      Object.assign(config, res.data)
      normalizePower(res.data.power)
    } else {
      normalizePower()
    }
  } catch (e) {
    normalizePower()
  }
}

const save = async () => {
  try {
    normalizePower(config.power)
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
