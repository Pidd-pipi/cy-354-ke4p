<template>
  <div class="page">
    <h2>🎓 毕业季专场</h2>
    <el-alert title="毕业季专场：学长学姐闲置好物集中放送" type="warning" :closable="false" class="banner" />
    <el-row :gutter="16">
      <el-col v-for="p in products" :key="p.id" :span="6" class="col">
        <ProductCard :product="p" @detail="showDetail" @buy="showDetail" />
      </el-col>
    </el-row>
    <el-empty v-if="!loading && products.length === 0" description="专场暂无商品" />
    <ProductDetailDialog
      v-model="detailVisible"
      :product-id="currentId"
      :is-owner="currentOwnerId === authStore.user?.id"
      @bought="reload"
      @chat="chatById"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import ProductCard from '../components/common/ProductCard.vue'
import ProductDetailDialog from '../components/common/ProductDetailDialog.vue'
import { listGraduation } from '../api/product'
import { createConversation } from '../api/conversation'
import type { Product } from '../types'
import { useAuthStore } from '../stores/authStore'
import { useRouter } from 'vue-router'

const products = ref<Product[]>([])
const loading = ref(false)
const detailVisible = ref(false)
const currentId = ref<number | null>(null)
const currentOwnerId = ref<number | null>(null)
const authStore = useAuthStore()
const router = useRouter()

function showDetail(p: Product) {
  currentId.value = p.id
  currentOwnerId.value = p.seller_id
  detailVisible.value = true
}

async function chatById(productId: number) {
  if (!authStore.token) {
    ElMessage.warning('请先登录')
    router.push('/login')
    return
  }
  await createConversation(productId)
  ElMessage.success('已发起私信')
  router.push('/messages')
}

async function reload() {
  loading.value = true
  try {
    const res = await listGraduation()
    products.value = res.data.items
  } finally {
    loading.value = false
  }
}

onMounted(reload)
</script>

<style scoped>
.banner {
  margin-bottom: 16px;
}
.col {
  margin-bottom: 16px;
}
</style>
