<template>
  <el-dialog :model-value="modelValue" :title="detail?.product.title" width="560px" @update:model-value="$emit('update:modelValue', $event)" @close="$emit('update:modelValue', false)">
    <el-descriptions v-if="detail" :column="2" border>
      <el-descriptions-item label="分类">{{ categoryLabel(detail.product.category) }}</el-descriptions-item>
      <el-descriptions-item label="成色">{{ detail.product.condition }}</el-descriptions-item>
      <el-descriptions-item label="校区">{{ detail.product.campus }}</el-descriptions-item>
      <el-descriptions-item label="交易地点">{{ detail.product.trade_location }}</el-descriptions-item>
      <el-descriptions-item label="价格">¥{{ detail.product.price.toFixed(2) }}</el-descriptions-item>
      <el-descriptions-item label="状态">{{ productStatusLabel(detail.product.status) }}</el-descriptions-item>
      <el-descriptions-item label="描述" :span="2">{{ detail.product.description }}</el-descriptions-item>
    </el-descriptions>

    <div v-if="detail && detail.total_slots > 0" class="slots-block">
      <div class="slots-head">
        <span>面交时段</span>
        <el-tag size="small" :type="detail.open_slots > 0 ? 'success' : 'info'">
          剩余可选 {{ detail.open_slots }} / {{ detail.total_slots }}
        </el-tag>
      </div>
      <el-radio-group v-model="selectedSlot" class="slots-group">
        <el-radio-button
          v-for="s in detail.slots"
          :key="s.id"
          :value="s.start_at"
          :disabled="!s.available"
        >
          {{ formatSlotRange(s.start_at) }}
          <span v-if="!s.available" class="slot-state">（{{ slotStateLabel(s) }}）</span>
        </el-radio-button>
      </el-radio-group>
      <el-empty v-if="detail.open_slots === 0" description="暂无可预约的时段" :image-size="50" />
    </div>

    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">关闭</el-button>
      <el-button
        v-if="canBuy"
        type="primary"
        :loading="booking"
        :disabled="requiresSlot && !selectedSlot"
        @click="confirmBook"
      >
        {{ requiresSlot ? '预约并下单' : '直接购买' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { categoryLabel, productStatusLabel } from '../../constants/product'
import { productSlotStatusLabel } from '../../constants/slot'
import { formatSlotRange } from '../../utils/dateFormat'
import type { ProductDetail, ProductSlot } from '../../types'

const props = defineProps<{
  modelValue: boolean
  detail: ProductDetail | null
  booking?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'book', slotStart?: string): void
}>()

const selectedSlot = ref('')
const booking = computed(() => props.booking ?? false)

watch(
  () => props.modelValue,
  (open) => {
    if (open) selectedSlot.value = ''
  },
)

// Products that publish slots require choosing one; slot-free products keep
// the legacy direct-order flow.
const requiresSlot = computed(() => !!props.detail && props.detail.total_slots > 0)
const canBuy = computed(() => !!props.detail && props.detail.product.status === 'on_sale' && props.detail.bookable)

function slotStateLabel(s: ProductSlot): string {
  if (s.status === 'open') return '已过期'
  return productSlotStatusLabel(s.status)
}

function confirmBook() {
  emit('book', requiresSlot.value ? selectedSlot.value : undefined)
}
</script>

<style scoped>
.slots-block {
  margin-top: 16px;
}
.slots-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  font-weight: 600;
}
.slots-group {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.slot-state {
  font-size: 12px;
  color: #909399;
}
</style>
