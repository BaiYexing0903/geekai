<template>
  <div class="welcome">
    <div class="container">
      <h2 class="title">{{ title }}-{{ version }}</h2>

    </div>
  </div>
</template>
<script setup>
import { getSystemInfo } from '@/store/cache'
import { ElMessage } from 'element-plus'
import { onMounted, ref } from 'vue'

const title = ref(import.meta.env.VITE_TITLE)
const version = ref(import.meta.env.VITE_VERSION)


onMounted(() => {
  getSystemInfo()
    .then((res) => {
      title.value = res.data.title
    })
    .catch((e) => {
      ElMessage.error('获取系统配置失败：' + e.message)
    })
})

const emits = defineEmits(['send'])
const send = (text) => {
  emits('send', text)
}
</script>
<style scoped lang="scss">
.welcome {
  text-align: center;
  display: flex;
  justify-content: center;
  margin-top: 8vh;

  .container {
    max-width: 768px;
    width: 100%;

    .title {
      // font-size: 2.25rem
      line-height: 2.5rem;
      font-weight: 600;
      margin-bottom: 4rem;
      color: var(--text-color);
    }

  }
}
</style>
