<template>
  <div class="page">
    <h2>发布商品</h2>
    <el-card style="max-width: 640px">
      <ProductForm ref="formRef" />
      <el-button type="primary" :loading="submitting" @click="submit">发布</el-button>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import ProductForm from '../components/common/ProductForm.vue'
import { createProduct } from '../api/product'
import { useAuthStore } from '../stores/authStore'
import { useRouter } from 'vue-router'

const formRef = ref<InstanceType<typeof ProductForm>>()
const submitting = ref(false)
const authStore = useAuthStore()
const router = useRouter()

async function submit() {
  if (!authStore.token) {
    ElMessage.warning('请先登录')
    router.push('/login')
    return
  }
  const form = formRef.value?.form
  if (!form || !form.title || !form.category || !form.campus || form.price <= 0) {
    ElMessage.warning('请填写完整信息')
    return
  }
  const { slots, incomplete } = formRef.value?.collectSlotStarts() ?? { slots: [], incomplete: false }
  if (incomplete) {
    ElMessage.warning('面交时段需选择未来的日期与整半小时时间，请检查后移除无效时段')
    return
  }
  submitting.value = true
  try {
    await createProduct({
      title: form.title,
      description: form.description,
      price: form.price,
      category: form.category,
      condition: form.condition,
      campus: form.campus,
      trade_location: form.trade_location,
      images: form.images,
      slot_starts: slots,
    })
    ElMessage.success(slots.length > 0 ? `发布成功，已开放 ${slots.length} 个面交时段` : '发布成功')
    router.push('/products')
  } finally {
    submitting.value = false
  }
}
</script>
