<template>
  <div class="slot-list">
    <div class="slot-summary">
      <el-tag type="success" size="small">剩余 {{ availableSlots }} / {{ slots.length }} 个可选时段</el-tag>
    </div>
    <el-radio-group v-if="availableSlots > 0" v-model="selected" class="slot-radios">
      <el-radio-button
        v-for="s in bookableSlots"
        :key="s.id"
        :value="s.id"
        class="slot-radio"
      >
        {{ formatSlotRange(s.start_time, s.end_time) }}
      </el-radio-button>
    </el-radio-group>
    <el-text v-else type="warning" size="small">暂无可用时段（可能已全部约满或过期）</el-text>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { formatSlotRange } from '../../utils/dateFormat'
import type { TradeSlot } from '../../types'

const props = defineProps<{ slots: TradeSlot[] }>()
const selected = ref<number | null>(null)

const bookableSlots = computed(() => props.slots.filter((s) => s.available))
const availableSlots = computed(() => bookableSlots.value.length)

watch(
  () => props.slots,
  (list) => {
    if (selected.value && !list.some((s) => s.id === selected.value && s.available)) {
      selected.value = null
    }
  },
)

defineExpose({ selected })
</script>

<style scoped>
.slot-summary {
  margin-bottom: 8px;
}
.slot-radios {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.slot-radio {
  margin: 0;
}
</style>
