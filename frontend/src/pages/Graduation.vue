<template>
  <div class="page">
    <h2>🎓 毕业季专场</h2>
    <el-alert title="毕业季专场：学长学姐闲置好物集中放送" type="warning" :closable="false" class="banner" />
    <el-row :gutter="16">
      <el-col v-for="p in products" :key="p.id" :span="6" class="col">
        <ProductCard :product="p" @detail="showDetail" @buy="buy" />
      </el-col>
    </el-row>
    <el-empty v-if="!loading && products.length === 0" description="专场暂无商品" />
    <ProductDetailDialog v-model="detailVisible" :detail="detail" :booking="booking" @book="bookSlot" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import ProductCard from '../components/common/ProductCard.vue'
import ProductDetailDialog from '../components/common/ProductDetailDialog.vue'
import { getProduct, listGraduation } from '../api/product'
import { createTradeOrder } from '../api/tradeOrder'
import type { Product, ProductDetail } from '../types'
import { useAuthStore } from '../stores/authStore'
import { useRouter } from 'vue-router'

const products = ref<Product[]>([])
const loading = ref(false)
const detailVisible = ref(false)
const detail = ref<ProductDetail | null>(null)
const booking = ref(false)
const current = ref<Product | null>(null)
const authStore = useAuthStore()
const router = useRouter()

async function showDetail(p: Product) {
  current.value = p
  detail.value = null
  detailVisible.value = true
  const res = await getProduct(p.id)
  detail.value = res.data
}

function buy(p: Product) {
  if (!authStore.token) {
    ElMessage.warning('请先登录')
    router.push('/login')
    return
  }
  showDetail(p)
}

async function bookSlot(slotStart?: string) {
  if (!current.value) return
  booking.value = true
  try {
    await createTradeOrder(current.value.id, slotStart)
    ElMessage.success(slotStart ? '时段预约成功，等待卖家确认' : '已下单')
    detailVisible.value = false
  } finally {
    booking.value = false
  }
}

onMounted(async () => {
  loading.value = true
  try {
    const res = await listGraduation()
    products.value = res.data.items
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.banner {
  margin-bottom: 16px;
}
.col {
  margin-bottom: 16px;
}
</style>
