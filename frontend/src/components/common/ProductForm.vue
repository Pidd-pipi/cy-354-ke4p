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
      <div class="slot-picker">
        <div class="slot-add">
          <el-date-picker
            v-model="slotDraft"
            type="datetime"
            placeholder="选择整点/半点时间"
            format="MM-DD HH:mm"
            value-format="YYYY-MM-DDTHH:mm:ss"
            :disabled-date="disabledPastDate"
            :clearable="false"
            class="slot-draft"
          />
          <el-button type="primary" plain @click="addSlot">添加半小时时段</el-button>
        </div>
        <el-text size="small" type="info">可选。不设置时段时买家可直接下单；设置后买家必须选择一个未过期、未被占用的时段。</el-text>
        <div class="slot-tags">
          <el-tag
            v-for="(s, i) in form.slots"
            :key="s.start_time + '_' + i"
            closable
            type="success"
            class="slot-tag"
            @close="removeSlot(i)"
          >
            {{ formatDateTime(s.start_time) }} - {{ formatDateTime(slotEnd(s.start_time)) }}
          </el-tag>
        </div>
      </div>
    </el-form-item>
  </el-form>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { PRODUCT_CATEGORIES } from '../../constants/product'
import { formatDateTime } from '../../utils/dateFormat'

export interface ProductFormValue {
  title: string
  description: string
  price: number
  category: string
  condition: string
  campus: string
  trade_location: string
  images: string
  slots: { start_time: string }[]
}

const form = reactive<ProductFormValue>({
  title: '',
  description: '',
  price: 0,
  category: '',
  condition: '',
  campus: '',
  trade_location: '',
  images: '',
  slots: [],
})

const slotDraft = ref<Date | string | null>(null)

function slotEnd(startISO: string): string {
  const d = new Date(startISO)
  d.setMinutes(d.getMinutes() + 30)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function disabledPastDate(date: Date): boolean {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return date.getTime() < today.getTime()
}

function addSlot() {
  if (!slotDraft.value) return
  const picked = new Date(slotDraft.value)
  if (Number.isNaN(picked.getTime())) return
  // snap to the half-hour grid server enforces
  const snapped = new Date(picked)
  snapped.setSeconds(0, 0)
  const min = snapped.getMinutes()
  if (min !== 0 && min !== 30) {
    snapped.setMinutes(min < 30 ? 30 : 0)
    if (min >= 30) snapped.setHours(snapped.getHours() + 1)
  }
  if (snapped.getTime() <= Date.now()) {
    slotDraft.value = null
    return
  }
  const iso = snapped.toISOString()
  if (form.slots.some((s) => s.start_time === iso)) {
    slotDraft.value = null
    return
  }
  form.slots.push({ start_time: iso })
  form.slots.sort((a, b) => +new Date(a.start_time) - +new Date(b.start_time))
  slotDraft.value = null
}

function removeSlot(index: number) {
  form.slots.splice(index, 1)
}

defineExpose({ form })
</script>

<style scoped>
.slot-picker {
  width: 100%;
}
.slot-add {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 6px;
}
.slot-draft {
  width: 200px;
}
.slot-tags {
  margin-top: 8px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.slot-tag {
  margin: 0;
}
</style>
