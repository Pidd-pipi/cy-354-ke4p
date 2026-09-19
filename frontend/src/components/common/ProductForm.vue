<template>
  <el-form :model="form" label-width="90px">
    <el-form-item label="标题">
      <el-input v-model="form.title" placeholder="商品标题" maxlength="64" />
    </el-form-item>
    <el-form-item label="描述">
      <el-input v-model="form.description" type="textarea" :rows="3" placeholder="描述商品" />
    </el-form-item>
    <el-form-item label="价格">
      <el-input-number v-model="form.price" :min="0.01" :precision="2" />
    </el-form-item>
    <el-form-item label="分类">
      <el-select v-model="form.category" placeholder="选择分类" style="width: 100%">
        <el-option v-for="c in PRODUCT_CATEGORIES" :key="c.value" :label="c.label" :value="c.value" />
      </el-select>
    </el-form-item>
    <el-form-item label="成色">
      <el-input v-model="form.condition" placeholder="如：全新/九成新" />
    </el-form-item>
    <el-form-item label="校区">
      <el-input v-model="form.campus" placeholder="如：东校区" />
    </el-form-item>
    <el-form-item label="交易地点">
      <el-input v-model="form.trade_location" placeholder="如：图书馆门口" />
    </el-form-item>
    <el-form-item label="实拍图URL">
      <el-input v-model="form.images" placeholder="可选，多个用逗号分隔" />
    </el-form-item>
    <el-form-item label="面交时段">
      <div class="slots-editor">
        <div class="slots-tip">可选。为商品添加半小时面交时段，买家下单时必须预约其中一个未过期的时段。</div>
        <div v-for="(s, idx) in form.slot_dates" :key="idx" class="slot-row">
          <el-date-picker
            v-model="s.date"
            type="date"
            placeholder="选择日期"
            value-format="x"
            :disabled-date="disablePastDate"
            style="width: 150px"
          />
          <el-select v-model="s.time" placeholder="时间" style="width: 110px">
            <el-option v-for="t in HALF_HOUR_OPTIONS" :key="t" :label="t" :value="t" />
          </el-select>
          <el-button text type="danger" @click="removeSlot(idx)">移除</el-button>
        </div>
        <el-button size="small" :disabled="form.slot_dates.length >= 30" @click="addSlot">+ 添加时段</el-button>
      </div>
    </el-form-item>
  </el-form>
</template>

<script setup lang="ts">
import { reactive } from 'vue'
import { PRODUCT_CATEGORIES } from '../../constants/product'
import { toRFC3339Local } from '../../constants/slot'

export interface SlotDraft {
  date: number | null
  time: string
}

export interface ProductFormValue {
  title: string
  description: string
  price: number
  category: string
  condition: string
  campus: string
  trade_location: string
  images: string
  slot_dates: SlotDraft[]
}

const HALF_HOUR_OPTIONS: string[] = (() => {
  const out: string[] = []
  for (let h = 0; h < 24; h++) {
    for (const m of [0, 30]) {
      out.push(`${String(h).padStart(2, '0')}:${m === 0 ? '00' : '30'}`)
    }
  }
  return out
})()

const form = reactive<ProductFormValue>({
  title: '',
  description: '',
  price: 0,
  category: '',
  condition: '',
  campus: '',
  trade_location: '',
  images: '',
  slot_dates: [],
})

function disablePastDate(d: Date): boolean {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return d.getTime() < today.getTime()
}

function addSlot() {
  form.slot_dates.push({ date: null, time: '10:00' })
}

function removeSlot(idx: number) {
  form.slot_dates.splice(idx, 1)
}

// Collects valid, future, half-hour aligned slots as RFC3339 strings,
// de-duplicated by timestamp. Returns the slot list plus rejected drafts.
function collectSlotStarts(now: Date = new Date()): { slots: string[]; incomplete: boolean } {
  const seen = new Set<number>()
  const slots: string[] = []
  let incomplete = false
  for (const s of form.slot_dates) {
    if (s.date === null || !s.time) {
      incomplete = true
      continue
    }
    const [hh, mm] = s.time.split(':').map(Number)
    const d = new Date(s.date)
    d.setHours(hh, mm, 0, 0)
    if (d.getTime() <= now.getTime()) {
      incomplete = true
      continue
    }
    if (seen.has(d.getTime())) continue
    seen.add(d.getTime())
    slots.push(toRFC3339Local(d))
  }
  slots.sort()
  return { slots, incomplete }
}

defineExpose({ form, addSlot, removeSlot, collectSlotStarts })
</script>

<style scoped>
.slots-editor {
  width: 100%;
}
.slots-tip {
  font-size: 12px;
  color: #909399;
  margin-bottom: 8px;
}
.slot-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
}
</style>
