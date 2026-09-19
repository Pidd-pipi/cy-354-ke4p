<template>
  <div class="page">
    <h2>我的交易</h2>
    <el-card v-for="o in orders" :key="o.id" class="order-card">
      <div class="order-row">
        <div>
          <TradeStatusBadge :status="o.status" />
          <span class="order-id">订单 #{{ o.id }} · 商品 #{{ o.product_id }}</span>
          <p class="order-meta">
            买家 #{{ o.buyer_id }} / 卖家 #{{ o.seller_id }} · 下单 {{ formatDateTime(o.created_at) }}
          </p>
          <p v-if="o.slot" class="order-slot">
            <el-tag size="small" :type="o.status === 'cancelled' ? 'info' : 'warning'">
              面交时间 {{ formatSlotRange(o.slot.start_time, o.slot.end_time) }}
            </el-tag>
            <span class="slot-state">{{ slotStatusLabel(o.slot.status) }}{{ o.slot.expired ? '（已过期）' : '' }}</span>
          </p>
        </div>
        <div class="order-actions">
          <el-button v-if="o.status === 'pending' && o.buyer_id === authStore.user?.id" size="small" type="primary" @click="onBuyerConfirm(o.id)">确认收货</el-button>
          <el-button v-if="o.status === 'confirmed' && o.seller_id === authStore.user?.id" size="small" type="success" @click="onSellerConfirm(o.id)">确认收款</el-button>
          <el-button v-if="o.status === 'pending'" size="small" type="danger" @click="onCancel(o.id)">取消</el-button>
          <el-button v-if="o.status === 'completed'" size="small" @click="reviewDialog(o)">评价</el-button>
        </div>
      </div>
    </el-card>
    <el-empty v-if="orders.length === 0" description="暂无交易" />
    <el-dialog v-model="reviewVisible" title="信誉评价" width="420px">
      <el-form label-width="70px">
        <el-form-item label="评价">
          <el-select v-model="reviewForm.rating" style="width: 100%">
            <el-option v-for="r in REVIEW_RATINGS" :key="r.value" :label="r.label" :value="r.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="内容">
          <el-input v-model="reviewForm.content" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="reviewVisible = false">取消</el-button>
        <el-button type="primary" @click="submitReview">提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import TradeStatusBadge from '../components/common/TradeStatusBadge.vue'
import { useTradeStore } from '../stores/tradeStore'
import { useAuthStore } from '../stores/authStore'
import { buyerConfirm, sellerConfirm, cancelTradeOrder } from '../api/tradeOrder'
import { createReview } from '../api/review'
import { REVIEW_RATINGS } from '../constants/trade'
import { slotStatusLabel } from '../constants/slot'
import { formatDateTime, formatSlotRange } from '../utils/dateFormat'
import type { TradeOrder } from '../types'

const { orders, fetch } = useTradeStore()
const authStore = useAuthStore()
const reviewVisible = ref(false)
const reviewForm = reactive({ trade_id: 0, rating: 'good', content: '' })

async function onBuyerConfirm(id: number) {
  await buyerConfirm(id)
  ElMessage.success('已确认收货，等待卖家确认收款')
  await fetch()
}

async function onSellerConfirm(id: number) {
  await sellerConfirm(id)
  ElMessage.success('交易完成，面交时段已锁定')
  await fetch()
}

async function onCancel(id: number) {
  try {
    await ElMessageBox.confirm('取消后该面交时段将被释放，其他买家可重新预约。确认取消该订单？', '取消订单', {
      type: 'warning',
      confirmButtonText: '确认取消',
      cancelButtonText: '再想想',
    })
  } catch {
    return
  }
  await cancelTradeOrder(id)
  ElMessage.success('已取消，面交时段已释放')
  await fetch()
}

function reviewDialog(o: TradeOrder) {
  reviewForm.trade_id = o.id
  reviewForm.rating = 'good'
  reviewForm.content = ''
  reviewVisible.value = true
}

async function submitReview() {
  await createReview({ trade_id: reviewForm.trade_id, rating: reviewForm.rating, content: reviewForm.content })
  ElMessage.success('评价成功')
  reviewVisible.value = false
}

onMounted(fetch)
</script>

<style scoped>
.order-card {
  margin-bottom: 12px;
}
.order-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.order-id {
  margin-left: 8px;
  font-size: 13px;
  color: #606266;
}
.order-meta {
  color: #909399;
  font-size: 12px;
  margin: 6px 0 0;
}
.order-slot {
  margin: 8px 0 0;
  display: flex;
  align-items: center;
  gap: 8px;
}
.slot-state {
  font-size: 12px;
  color: #909399;
}
</style>
