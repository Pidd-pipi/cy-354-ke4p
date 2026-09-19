package constants

// Messages centralizes user-facing prompts, backend responses and log wording.
const (
	MsgOK                 = "ok"
	MsgValidationFailed   = "参数校验失败"
	MsgUnauthorized       = "未登录或登录已过期"
	MsgForbidden          = "没有操作权限"
	MsgNotFound           = "资源不存在"
	MsgConflict           = "资源状态冲突"
	MsgRateLimited        = "请求过于频繁，请稍后再试"
	MsgInternalError      = "服务器内部错误"
	MsgPhoneAlreadyUsed   = "该手机号已被注册"
	MsgPhoneOrPassword    = "手机号或密码错误"
	MsgProductNotOnSale   = "商品当前不在售"
	MsgProductSold        = "商品已售出"
	MsgSlotRequired       = "该商品为预约面交商品，请选择面交时段"
	MsgSlotUnavailable    = "该面交时段已过期或已被占用，请选择其他时段"
	MsgSlotExpired        = "不能选择已过期的面交时段"
	MsgSlotInvalid        = "面交时段不合法（需为未过期的整半小时时段）"
	MsgSlotProductNoSlots = "该商品未开放面交预约"
	MsgSlotLimitExceeded  = "面交时段数量超出限制"
	MsgTradeStatusInvalid = "当前交易状态不可操作"
	MsgNotParticipant     = "仅买家和卖家可操作该订单"
	MsgAlreadyReviewed    = "该交易已评价"
	MsgExchangeClosed     = "该交换已关闭"
	MsgNoMatch            = "暂未找到匹配的书籍交换"
	MsgReportTarget       = "举报对象无效"
)
