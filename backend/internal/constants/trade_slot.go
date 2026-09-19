package constants

import "time"

// SlotStatus defines face-to-face handover slot states shared with the frontend.
const (
	SlotStatusAvailable = "available" // 可预约
	SlotStatusLocked    = "locked"    // 已被订单占用（待确认/已确认/已完成）
	SlotStatusReleased  = "released"  // 待确认订单取消后释放（可被再次预约）
)

// SlotStatuses lists all valid slot statuses.
var SlotStatuses = []string{
	SlotStatusAvailable, SlotStatusLocked, SlotStatusReleased,
}

// IsSlotStatus reports whether the given slot status is valid.
func IsSlotStatus(s string) bool {
	for _, v := range SlotStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// SlotDuration is the fixed length of every handover window.
const SlotDuration = 30 * time.Minute

// MaxSlotsPerProduct bounds how many slots a seller may attach to one product.
const MaxSlotsPerProduct = 40

// SlotStatusText returns the Chinese label of a slot status.
func SlotStatusText(s string) string {
	switch s {
	case SlotStatusAvailable:
		return "可预约"
	case SlotStatusLocked:
		return "已预约"
	case SlotStatusReleased:
		return "已释放"
	default:
		return "未知"
	}
}
