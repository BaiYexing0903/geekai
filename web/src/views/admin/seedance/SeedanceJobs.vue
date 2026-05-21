<template>
  <div class="seedance-jobs">
    <!-- 统计卡片 -->
    <div class="stats-row">
      <div class="stat-card"><div class="stat-num">{{ stats.total }}</div><div class="stat-label">总任务</div></div>
      <div class="stat-card processing"><div class="stat-num">{{ stats.processing || 0 }}</div><div class="stat-label">处理中</div></div>
      <div class="stat-card success"><div class="stat-num">{{ stats.succeeded || 0 }}</div><div class="stat-label">已完成</div></div>
      <div class="stat-card failed"><div class="stat-num">{{ stats.failed || 0 }}</div><div class="stat-label">失败</div></div>
    </div>

    <!-- 筛选 -->
    <div class="filter-bar">
      <el-input v-model="filters.user_id" placeholder="用户ID" size="small" style="width: 120px" clearable />
      <el-select v-model="filters.type" placeholder="任务类型" size="small" clearable style="width: 150px">
        <el-option v-for="m in modeOptions" :key="m.value" :label="m.label" :value="m.value" />
      </el-select>
      <el-select v-model="filters.status" placeholder="状态" size="small" clearable style="width: 120px">
        <el-option label="排队中" value="queued" />
        <el-option label="生成中" value="running" />
        <el-option label="已完成" value="succeeded" />
        <el-option label="失败" value="failed" />
        <el-option label="已过期" value="expired" />
      </el-select>
      <el-button type="primary" size="small" @click="fetchJobs">搜索</el-button>
      <el-button type="danger" size="small" @click="batchDelete" :disabled="selectedIds.length === 0">批量删除</el-button>
    </div>

    <!-- 表格 -->
    <el-table :data="jobs" @selection-change="handleSelection" style="width: 100%">
      <el-table-column type="selection" width="50" />
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="user_id" label="用户ID" width="80" />
      <el-table-column prop="type" label="模式" width="130">
        <template #default="{ row }">
          <el-tag size="small">{{ getModeName(row.type) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="prompt" label="提示词" min-width="200" show-overflow-tooltip />
      <el-table-column prop="status" label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)" size="small">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="power" label="算力" width="70" />
      <el-table-column prop="created_at" label="创建时间" width="170">
        <template #default="{ row }">{{ new Date(row.created_at * 1000).toLocaleString() }}</template>
      </el-table-column>
      <el-table-column label="操作" width="80">
        <template #default="{ row }">
          <el-button size="small" type="danger" text @click="removeJob(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      v-model:current-page="page"
      :page-size="pageSize"
      :total="total"
      layout="total, prev, pager, next"
      @current-change="fetchJobs"
      style="margin-top: 16px; justify-content: flex-end"
    />
  </div>
</template>

<script setup>
import { httpGet, httpPost } from '@/utils/http'
import { showMessageOK, showMessageError } from '@/utils/dialog'
import { ElMessageBox } from 'element-plus'
import { onMounted, reactive, ref } from 'vue'

const jobs = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const stats = reactive({ total: 0, processing: 0, succeeded: 0, failed: 0 })
const selectedIds = ref([])
const filters = reactive({ user_id: '', type: '', status: '' })

const modeOptions = [
  { label: '文生视频', value: 'text_to_video' },
  { label: '图生视频-首帧', value: 'image_to_video_first' },
  { label: '图生视频-首尾帧', value: 'image_to_video_dual' },
  { label: '多模态参考', value: 'multimodal_ref' },
  { label: '编辑视频', value: 'edit_video' },
  { label: '延长视频', value: 'extend_video' },
  { label: '虚拟人像', value: 'virtual_avatar' },
]

const modeMap = Object.fromEntries(modeOptions.map((m) => [m.value, m.label]))
const getModeName = (type) => modeMap[type] || type
const statusType = (s) => ({ succeeded: 'success', failed: 'danger', queued: 'info', running: 'warning' }[s] || 'info')

const fetchJobs = async () => {
  try {
    const params = new URLSearchParams({ page: page.value, page_size: pageSize })
    if (filters.user_id) params.set('user_id', filters.user_id)
    if (filters.type) params.set('type', filters.type)
    if (filters.status) params.set('status', filters.status)
    const res = await httpGet(`/api/admin/seedance/jobs?${params}`)
    jobs.value = res.data?.items || []
    total.value = res.data?.total || 0
  } catch (e) {
    showMessageError('获取列表失败')
  }
}

const fetchStats = async () => {
  try {
    const res = await httpGet('/api/admin/seedance/stats')
    if (res.data) Object.assign(stats, res.data)
  } catch (e) { /* ignore */ }
}

const handleSelection = (rows) => {
  selectedIds.value = rows.map((r) => r.id)
}

const batchDelete = async () => {
  try {
    await ElMessageBox.confirm(`确定删除 ${selectedIds.value.length} 个任务？`, '提示', { type: 'warning' })
    await httpPost('/api/admin/seedance/jobs/remove', { ids: selectedIds.value })
    showMessageOK('删除成功')
    fetchJobs()
    fetchStats()
  } catch (e) { /* cancelled */ }
}

const removeJob = async (row) => {
  try {
    await ElMessageBox.confirm('确定删除该任务？', '提示', { type: 'warning' })
    await httpPost('/api/admin/seedance/jobs/remove', { ids: [row.id] })
    showMessageOK('删除成功')
    fetchJobs()
    fetchStats()
  } catch (e) { /* cancelled */ }
}

onMounted(() => { fetchJobs(); fetchStats() })
</script>

<style scoped>
.seedance-jobs { padding: 20px; }
.stats-row { display: flex; gap: 12px; margin-bottom: 16px; }
.stat-card {
  background: #fff; border-radius: 8px; padding: 16px; text-align: center; flex: 1;
  border-left: 3px solid #409eff;
}
.stat-card.processing { border-color: #e6a23c; }
.stat-card.success { border-color: #67c23a; }
.stat-card.failed { border-color: #f56c6c; }
.stat-num { font-size: 24px; font-weight: 600; }
.stat-label { font-size: 12px; color: #999; margin-top: 4px; }
.filter-bar { display: flex; gap: 8px; margin-bottom: 16px; align-items: center; flex-wrap: wrap; }
</style>
