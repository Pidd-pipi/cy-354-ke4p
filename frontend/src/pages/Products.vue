<template>
  <div class="page">
    <h2>商品广场</h2>
    <el-form inline class="filters">
      <el-form-item label="分类">
        <el-select v-model="query.category" clearable placeholder="全部分类" style="width: 160px">
          <el-option v-for="c in PRODUCT_CATEGORIES" :key="c.value" :label="c.label" :value="c.value" />
        </el-select>
      </el-form-item>
      <el-form-item label="校区">
        <el-input v-model="query.campus" placeholder="输入校区" style="width: 140px" clearable />
      </el-form-item>
      <el-form-item label="关键词">
        <el-input v-model="query.keyword" placeholder="搜索标题/描述" style="width: 180px" clearable />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="load">搜索</el-button>
      </el-form-item>
    </el-form>
    <el-row :gutter="16">
      <el-col v-for="p in products" :key="p.id" :span="6" class="col">
        <ProductCard :product="p" @detail="showDetail" @buy="buy" @chat="chat" />
      </el-col>
    </el-row>
    <el-empty v-if="!loading && products.length === 0" description="暂无商品" />
    <ProductDetailDialog v-model="detailVisible" :detail="detail" :booking="booking" @book="bookSlot" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import ProductCard from '../components/common/ProductCard.vue'
import ProductDetailDialog from '../components/common/ProductDetailDialog.vue'
import { PRODUCT_CATEGORIES } from '../constants/product'
import { useProducts } from '../hooks/useProducts'
import { getProduct } from '../api/product'
import { createTradeOrder } from '../api/tradeOrder'
import { createConversation } from '../api/conversation'
import type { Product, ProductDetail } from '../types'
import { useAuthStore } from '../stores/authStore'
import { useRouter } from 'vue-router'

const { products, loading, load } = useProducts()
const query = reactive<{ category?: string; campus?: string; keyword?: string }>({})
const detailVisible = ref(false)
const detail = ref<ProductDetail | null>(null)
const booking = ref(false)
const current = ref<Product | null>(null)
const authStore = useAuthStore()
const router = useRouter()

function ensureLogin(): boolean {
  if (!authStore.token) {
    ElMessage.warning('请先登录')
    router.push('/login')
    return false
  }
  return true
}

async function showDetail(p: Product) {
  current.value = p
  detail.value = null
  detailVisible.value = true
  const res = await getProduct(p.id)
  detail.value = res.data
}

function buy(p: Product) {
  if (!ensureLogin()) return
  showDetail(p)
}

async function bookSlot(slotStart?: string) {
  if (!current.value || !ensureLogin()) return
  booking.value = true
  try {
    await createTradeOrder(current.value.id, slotStart)
    ElMessage.success(slotStart ? '时段预约成功，等待卖家确认' : '已下单，等待卖家确认')
    detailVisible.value = false
    await load()
  } finally {
    booking.value = false
  }
}

async function chat(p: Product) {
  if (!ensureLogin()) return
  await createConversation(p.id)
  ElMessage.success('已发起私信')
  router.push('/messages')
}

onMounted(() => load())
</script>

<style scoped>
.filters {
  margin-bottom: 8px;
}
.col {
  margin-bottom: 16px;
}
</style>
