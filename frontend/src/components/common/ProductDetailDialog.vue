<template>
  <el-dialog
    :model-value="modelValue"
    :title="detail?.title ?? '商品详情'"
    width="560px"
    @update:model-value="(v: boolean) => emit('update:modelValue', v)"
  >
    <div v-loading="loading">
      <el-descriptions v-if="detail" :column="2" border>
        <el-descriptions-item label="分类">{{ categoryLabel(detail.category) }}</el-descriptions-item>
        <el-descriptions-item label="成色">{{ detail.condition }}</el-descriptions-item>
        <el-descriptions-item label="校区">{{ detail.campus }}</el-descriptions-item>
        <el-descriptions-item label="交易地点">{{ detail.trade_location }}</el-descriptions-item>
        <el-descriptions-item label="价格">¥{{ detail.price.toFixed(2) }}</el-descriptions-item>
        <el-descriptions-item label="状态">{{ productStatusLabel(detail.status) }}</el-descriptions-item>
        <el-descriptions-item label="描述" :span="2">{{ detail.description }}</el-descriptions-item>
      </el-descriptions>

      <div v-if="detail?.has_slots" class="slot-block">
        <div class="slot-title">面交时段</div>
        <ProductSlots ref="slotsRef" :slots="detail.slots" />
      </div>
    </div>
    <template #footer>
      <el-button @click="emit('update:modelValue', false)">关闭</el-button>
      <el-button :disabled="!canChat" @click="onChat">私信卖家</el-button>
      <el-button
        type="primary"
        :loading="buying"
        :disabled="!canBuy"
        @click="onBuy"
      >
        {{ buyLabel }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { getProduct } from '../../api/product'
import { createTradeOrder } from '../../api/tradeOrder'
import { categoryLabel, productStatusLabel } from '../../constants/product'
import ProductSlots from './ProductSlots.vue'
import { useAuthStore } from '../../stores/authStore'
import { useRouter } from 'vue-router'
import type { ProductDetail } from '../../types'

const props = defineProps<{
  modelValue: boolean
  productId: number | null
  isOwner?: boolean
}>()
const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'bought'): void
  (e: 'chat', productId: number): void
}>()

const authStore = useAuthStore()
const router = useRouter()
const detail = ref<ProductDetail | null>(null)
const loading = ref(false)
const buying = ref(false)
const slotsRef = ref<InstanceType<typeof ProductSlots>>()

async function load() {
  if (props.productId == null) return
  loading.value = true
  detail.value = null
  try {
    const res = await getProduct(props.productId)
    detail.value = res.data
  } finally {
    loading.value = false
  }
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) load()
  },
)

const canBuy = computed(() => detail.value?.status === 'on_sale' && !props.isOwner)
const canChat = computed(() => !!detail.value && !props.isOwner)
const buyLabel = computed(() => {
  if (!detail.value) return '立即购买'
  if (detail.value.has_slots) return detail.value.available_slots > 0 ? '预约并下单' : '时段已满'
  return '立即购买'
})

function requireLogin(): boolean {
  if (!authStore.token) {
    ElMessage.warning('请先登录')
    router.push('/login')
    return false
  }
  return true
}

async function onBuy() {
  if (!detail.value || !requireLogin()) return
  let slotId: number | null = null
  if (detail.value.has_slots) {
    slotId = slotsRef.value?.selected ?? null
    if (slotId == null) {
      ElMessage.warning('请选择一个面交时段')
      return
    }
  }
  buying.value = true
  try {
    await createTradeOrder(detail.value.id, slotId)
    ElMessage.success(detail.value.has_slots ? '已预约面交时段，等待卖家确认' : '已下单，等待卖家确认')
    emit('update:modelValue', false)
    emit('bought')
  } finally {
    buying.value = false
  }
}

function onChat() {
  if (!detail.value || !requireLogin()) return
  emit('update:modelValue', false)
  emit('chat', detail.value.id)
}
</script>

<style scoped>
.slot-block {
  margin-top: 16px;
}
.slot-title {
  font-weight: 600;
  margin-bottom: 8px;
}
</style>
