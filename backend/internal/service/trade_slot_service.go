package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/lp/campus-market/internal/constants"
	"github.com/lp/campus-market/internal/dto"
	"github.com/lp/campus-market/internal/model"
	"github.com/lp/campus-market/internal/util"
)

// SlotRepository is the data access contract for handover slot rows.
type SlotRepository interface {
	CreateBatch(ctx context.Context, slots []model.TradeSlot) error
	ListByProduct(ctx context.Context, productID uint) ([]model.TradeSlot, error)
	FindByID(ctx context.Context, id uint) (*model.TradeSlot, error)
	TryOccupy(ctx context.Context, slotID, orderID uint, now time.Time) error
	Release(ctx context.Context, slotID uint, now time.Time) error
	PreloadOrders(ctx context.Context, orders []model.TradeOrder) error
	Transaction(ctx context.Context, fn func(txCtx context.Context) error) error
}

// TradeSlotService validates seller slot input and computes availability.
type TradeSlotService struct {
	slots  SlotRepository
	logger *slog.Logger
}

// NewTradeSlotService wires the trade slot service dependencies.
func NewTradeSlotService(slots SlotRepository, logger *slog.Logger) *TradeSlotService {
	return &TradeSlotService{slots: slots, logger: logger}
}

// BuildSlots converts publish-time input into slot models, enforcing the
// half-hour grid, future start times, uniqueness and the per-product cap.
func (s *TradeSlotService) BuildSlots(inputs []dto.CreateSlotInput, now time.Time) ([]model.TradeSlot, error) {
	if len(inputs) > constants.MaxSlotsPerProduct {
		return nil, util.NewAppError(400, constants.CodeBadRequest, constants.MsgSlotTooMany, nil)
	}
	seen := make(map[time.Time]struct{}, len(inputs))
	slots := make([]model.TradeSlot, 0, len(inputs))
	for _, in := range inputs {
		// JSON unmarshals RFC3339 timestamps into UTC, while the DSN parses
		// DATETIME columns with loc=Local. Re-anchor the wall-clock fields to
		// the server timezone so a 14:00 selection is stored and read back as
		// 14:00, and stays comparable with the DB NOW() used in TryOccupy.
		start := toLocalWallClock(in.StartTime)
		if m := start.Minute(); m != 0 && m != 30 {
			return nil, util.NewAppError(400, constants.CodeValidation, constants.MsgSlotInvalidTime, nil)
		}
		if !start.After(now) {
			return nil, util.NewAppError(400, constants.CodeValidation, constants.MsgSlotInvalidTime, nil)
		}
		if _, dup := seen[start]; dup {
			return nil, util.NewAppError(400, constants.CodeValidation, constants.MsgSlotDuplicate, nil)
		}
		seen[start] = struct{}{}
		slots = append(slots, model.TradeSlot{
			StartTime: start,
			EndTime:   start.Add(constants.SlotDuration),
			Status:    constants.SlotStatusAvailable,
		})
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i].StartTime.Before(slots[j].StartTime) })
	return slots, nil
}

// truncateMinute drops seconds and sub-second precision of a timestamp,
// preserving its location so the half-hour grid matches the client clock.
func truncateMinute(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location())
}

// toLocalWallClock keeps the calendar fields (year..minute) of t but labels
// them in the server's local timezone, then truncates to the minute.
func toLocalWallClock(t time.Time) time.Time {
	t = truncateMinute(t)
	local := t.In(time.Local)
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, local.Location())
}

// CreateBatch persists the given slots.
func (s *TradeSlotService) CreateBatch(ctx context.Context, slots []model.TradeSlot) error {
	if err := s.slots.CreateBatch(ctx, slots); err != nil {
		return util.WrapAppError(fmt.Errorf("trade_slot create batch: %w", err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	return nil
}

// ListByProduct returns the raw slots of a product.
func (s *TradeSlotService) ListByProduct(ctx context.Context, productID uint) ([]model.TradeSlot, error) {
	slots, err := s.slots.ListByProduct(ctx, productID)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("trade_slot[product=%d] list: %w", productID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	return slots, nil
}

// DetailView builds the slot section of a product-detail response. A slot is
// bookable when it is free (available/released) and not expired.
func (s *TradeSlotService) DetailView(ctx context.Context, productID uint, now time.Time) ([]dto.SlotView, int, error) {
	slots, err := s.ListByProduct(ctx, productID)
	if err != nil {
		return nil, 0, err
	}
	views := make([]dto.SlotView, 0, len(slots))
	available := 0
	for _, sl := range slots {
		expired := !sl.StartTime.After(now)
		free := sl.Status == constants.SlotStatusAvailable || sl.Status == constants.SlotStatusReleased
		bookable := free && !expired
		if bookable {
			available++
		}
		views = append(views, dto.SlotView{
			ID:        sl.ID,
			StartTime: sl.StartTime,
			EndTime:   sl.EndTime,
			Status:    sl.Status,
			Available: bookable,
			Expired:   expired,
		})
	}
	s.logger.Info(fmt.Sprintf(constants.LogTradeSlotListSuccess, productID, available, len(views)))
	return views, available, nil
}
