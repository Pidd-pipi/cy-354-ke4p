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
        <el-button type="primary" @click="load(query)">搜索</el-button>
      </el-form-item>
    </el-form>
    <el-row :gutter="16">
      <el-col v-for="p in products" :key="p.id" :span="6" class="col">
        <ProductCard
          :product="p"
          @detail="showDetail"
          @buy="showDetail"
          @chat="chat"
        />
      </el-col>
    </el-row>
    <el-empty v-if="!loading && products.length === 0" description="暂无商品" />
    <ProductDetailDialog
      v-model="detailVisible"
      :product-id="currentId"
      :is-owner="currentOwnerId === authStore.user?.id"
      @bought="load"
      @chat="chatById"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import ProductCard from '../components/common/ProductCard.vue'
import ProductDetailDialog from '../components/common/ProductDetailDialog.vue'
import { PRODUCT_CATEGORIES } from '../constants/product'
import { useProducts } from '../hooks/useProducts'
import { createConversation } from '../api/conversation'
import type { Product } from '../types'
import { useAuthStore } from '../stores/authStore'
import { useRouter } from 'vue-router'

const { products, loading, load } = useProducts()
const query = reactive<{ category?: string; campus?: string; keyword?: string }>({})
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

async function chat(p: Product) {
  await chatById(p.id)
}

onMounted(() => load(query))
</script>

<style scoped>
.filters {
  margin-bottom: 8px;
}
.col {
  margin-bottom: 16px;
}
</style>
