package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/lp/campus-market/internal/constants"
	"github.com/lp/campus-market/internal/dto"
	"github.com/lp/campus-market/internal/model"
	"github.com/lp/campus-market/internal/util"
)

// OrderRepository is the data access contract for trade order rows.
type OrderRepository interface {
	Create(ctx context.Context, o *model.TradeOrder) error
	FindByID(ctx context.Context, id uint) (*model.TradeOrder, error)
	FindByProductAndBuyer(ctx context.Context, productID, buyerID uint) (*model.TradeOrder, error)
	ListByUser(ctx context.Context, userID uint, page, pageSize int) ([]model.TradeOrder, int64, error)
	UpdateStatus(ctx context.Context, id uint, status string) error
	UpdateBuyerConfirmed(ctx context.Context, id uint, ts interface{}) error
	UpdateSellerConfirmed(ctx context.Context, id uint, ts interface{}) error
	Transaction(ctx context.Context, fn func(txCtx context.Context) error) error
}

// TradeOrderService manages purchase intents, confirmations and completion.
type TradeOrderService struct {
	orders   OrderRepository
	products ProductRepository
	slots    SlotRepository
	logger   *slog.Logger
}

// NewTradeOrderService wires the trade order service dependencies.
func NewTradeOrderService(orders OrderRepository, products ProductRepository, slots SlotRepository, logger *slog.Logger) *TradeOrderService {
	return &TradeOrderService{orders: orders, products: products, slots: slots, logger: logger}
}

// Create creates a pending trade order for an on-sale product, atomically
// reserving the chosen handover slot when the product defines any.
func (s *TradeOrderService) Create(ctx context.Context, buyer *model.User, req *dto.CreateTradeOrderRequest) (*model.TradeOrder, error) {
	product, err := s.products.FindByID(ctx, req.ProductID)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("trade_order[buyer=%d] product lookup: %w", buyer.ID, err), 404, constants.CodeNotFound, constants.MsgNotFound)
	}
	if product.SellerID == buyer.ID {
		return nil, util.NewAppError(400, constants.CodeBadRequest, "不能购买自己的商品", nil)
	}
	if product.Status != constants.ProductStatusOnSale {
		return nil, util.NewAppError(409, constants.CodeConflict, constants.MsgProductNotOnSale, nil)
	}
	if existing, err := s.orders.FindByProductAndBuyer(ctx, req.ProductID, buyer.ID); err == nil && existing != nil {
		return nil, util.NewAppError(409, constants.CodeConflict, "您已对该商品下单", nil)
	}

	productSlots, slotErr := s.slots.ListByProduct(ctx, req.ProductID)
	if slotErr != nil {
		return nil, util.WrapAppError(fmt.Errorf("trade_order[buyer=%d] slot lookup: %w", buyer.ID, slotErr), 500, constants.CodeInternalError, constants.MsgInternalError)
	}

	var targetSlot *model.TradeSlot
	if len(productSlots) > 0 {
		// Product defines handover windows: picking a valid, free slot is mandatory.
		if req.SlotID == nil {
			return nil, util.NewAppError(400, constants.CodeValidation, constants.MsgSlotRequired, nil)
		}
		slot, err := s.slots.FindByID(ctx, *req.SlotID)
		if err != nil {
			if errors.Is(err, util.ErrNotFound) {
				return nil, util.NewAppError(404, constants.CodeNotFound, constants.MsgSlotNotFound, nil)
			}
			return nil, util.WrapAppError(fmt.Errorf("trade_order[buyer=%d] slot find: %w", buyer.ID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
		}
		if slot.ProductID != req.ProductID {
			return nil, util.NewAppError(400, constants.CodeBadRequest, constants.MsgSlotProductMismatch, nil)
		}
		targetSlot = slot
	} else if req.SlotID != nil {
		return nil, util.NewAppError(400, constants.CodeBadRequest, constants.MsgSlotProductMismatch, nil)
	}

	var order *model.TradeOrder
	if targetSlot != nil {
		slotID := targetSlot.ID
		order = &model.TradeOrder{
			ProductID: req.ProductID, BuyerID: buyer.ID, SellerID: product.SellerID,
			SlotID: &slotID, Status: constants.TradeStatusPending,
		}
	} else {
		order = &model.TradeOrder{
			ProductID: req.ProductID, BuyerID: buyer.ID, SellerID: product.SellerID,
			Status: constants.TradeStatusPending,
		}
	}
	now := time.Now()

	if targetSlot != nil {
		// Reserve slot and create the order in one transaction. The atomic
		// conditional UPDATE guarantees that concurrent buyers racing for the
		// same half-hour window cannot both succeed: the loser rolls back and
		// no duplicate order is left behind.
		txErr := s.slots.Transaction(ctx, func(txCtx context.Context) error {
			if err := s.orders.Create(txCtx, order); err != nil {
				return fmt.Errorf("trade_order[buyer=%d] create: %w", buyer.ID, err)
			}
			if err := s.slots.TryOccupy(txCtx, targetSlot.ID, order.ID, now); err != nil {
				return err
			}
			return nil
		})
		if txErr != nil {
			if errors.Is(txErr, util.ErrConflict) {
				s.logger.Warn(fmt.Sprintf(constants.LogTradeSlotLockFailed, targetSlot.ID, buyer.ID, txErr))
				return nil, util.NewAppError(409, constants.CodeConflict, constants.MsgSlotTaken, nil)
			}
			return nil, util.WrapAppError(fmt.Errorf("trade_order[buyer=%d] reserve slot: %w", buyer.ID, txErr), 500, constants.CodeInternalError, constants.MsgInternalError)
		}
		order.Slot = &model.TradeSlot{
			ID: targetSlot.ID, ProductID: targetSlot.ProductID,
			StartTime: targetSlot.StartTime, EndTime: targetSlot.EndTime,
			Status: constants.SlotStatusLocked, OrderID: &order.ID,
		}
		s.logger.Info(fmt.Sprintf(constants.LogTradeSlotLockSuccess, targetSlot.ID, order.ID, req.ProductID))
	} else {
		if err := s.orders.Create(ctx, order); err != nil {
			return nil, util.WrapAppError(fmt.Errorf("trade_order[buyer=%d] create: %w", buyer.ID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
		}
	}

	s.logger.Info(fmt.Sprintf(constants.LogTradeOrderCreateSuccess, order.ID, req.ProductID))
	return order, nil
}

// ListMy returns the orders where the user participates, with slots attached.
func (s *TradeOrderService) ListMy(ctx context.Context, userID uint, q *dto.PageQuery) (*dto.PageResult, error) {
	q.Normalize()
	items, total, err := s.orders.ListByUser(ctx, userID, q.Page, q.PageSize)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("trade_order[user=%d] list: %w", userID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	if err := s.slots.PreloadOrders(ctx, items); err != nil {
		return nil, util.WrapAppError(fmt.Errorf("trade_order[user=%d] preload slots: %w", userID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	return &dto.PageResult{Items: items, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

// BuyerConfirm marks the order confirmed by the buyer. The slot stays locked.
func (s *TradeOrderService) BuyerConfirm(ctx context.Context, userID, orderID uint) (*model.TradeOrder, error) {
	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("trade_order[id=%d] buyer confirm find: %w", orderID, err), 404, constants.CodeNotFound, constants.MsgNotFound)
	}
	if order.BuyerID != userID {
		return nil, util.NewAppError(403, constants.CodeForbidden, constants.MsgNotParticipant, nil)
	}
	if order.Status != constants.TradeStatusPending {
		return nil, util.NewAppError(409, constants.CodeConflict, constants.MsgTradeStatusInvalid, nil)
	}
	now := time.Now()
	if err := s.orders.UpdateBuyerConfirmed(ctx, orderID, now); err != nil {
		return nil, util.WrapAppError(fmt.Errorf("trade_order[id=%d] buyer confirm: %w", orderID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	s.logger.Info(fmt.Sprintf(constants.LogTradeOrderBuyerConfirmSuccess, orderID))
	order.Status = constants.TradeStatusConfirmed
	s.attachSlot(ctx, order)
	return order, nil
}

// SellerConfirm completes the order and marks the product sold. The handover
// slot is permanently locked and is never released again.
func (s *TradeOrderService) SellerConfirm(ctx context.Context, userID, orderID uint) (*model.TradeOrder, error) {
	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("trade_order[id=%d] seller confirm find: %w", orderID, err), 404, constants.CodeNotFound, constants.MsgNotFound)
	}
	if order.SellerID != userID {
		return nil, util.NewAppError(403, constants.CodeForbidden, constants.MsgNotParticipant, nil)
	}
	if order.Status != constants.TradeStatusConfirmed {
		return nil, util.NewAppError(409, constants.CodeConflict, constants.MsgTradeStatusInvalid, nil)
	}
	now := time.Now()
	if err := s.orders.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.orders.UpdateSellerConfirmed(txCtx, orderID, now); err != nil {
			return err
		}
		if err := s.products.UpdateStatus(txCtx, order.ProductID, constants.ProductStatusSold); err != nil {
			return err
		}
		// Slot remains "locked" with order_id set: permanent lock, no release.
		return nil
	}); err != nil {
		if errors.Is(err, util.ErrConflict) {
			return nil, util.NewAppError(409, constants.CodeConflict, constants.MsgTradeStatusInvalid, nil)
		}
		s.logger.Error(fmt.Sprintf(constants.LogTradeOrderCompleteFailed, orderID, err))
		return nil, util.WrapAppError(fmt.Errorf("trade_order[id=%d] seller confirm: %w", orderID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	s.logger.Info(fmt.Sprintf(constants.LogTradeOrderCompleteSuccess, orderID, order.ProductID))
	order.Status = constants.TradeStatusCompleted
	s.attachSlot(ctx, order)
	return order, nil
}

// Cancel cancels a pending order. Either party may cancel while the order is
// still pending; the reserved handover slot (if any) is released so other
// buyers can book it. Confirmed or completed orders cannot be cancelled and
// keep their slot locked.
func (s *TradeOrderService) Cancel(ctx context.Context, userID, orderID uint) (*model.TradeOrder, error) {
	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("trade_order[id=%d] cancel find: %w", orderID, err), 404, constants.CodeNotFound, constants.MsgNotFound)
	}
	if order.BuyerID != userID && order.SellerID != userID {
		return nil, util.NewAppError(403, constants.CodeForbidden, constants.MsgNotParticipant, nil)
	}
	if order.Status != constants.TradeStatusPending {
		return nil, util.NewAppError(409, constants.CodeConflict, constants.MsgTradeStatusInvalid, nil)
	}
	now := time.Now()
	if order.SlotID != nil {
		if err := s.slots.Transaction(ctx, func(txCtx context.Context) error {
			if err := s.orders.UpdateStatus(txCtx, orderID, constants.TradeStatusCancelled); err != nil {
				return err
			}
			if err := s.slots.Release(txCtx, *order.SlotID, now); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return nil, util.WrapAppError(fmt.Errorf("trade_order[id=%d] cancel release: %w", orderID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
		}
		s.logger.Info(fmt.Sprintf(constants.LogTradeSlotReleaseSuccess, *order.SlotID, orderID))
	} else if err := s.orders.UpdateStatus(ctx, orderID, constants.TradeStatusCancelled); err != nil {
		return nil, util.WrapAppError(fmt.Errorf("trade_order[id=%d] cancel: %w", orderID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	s.logger.Info(fmt.Sprintf(constants.LogTradeOrderCancelSuccess, orderID))
	order.Status = constants.TradeStatusCancelled
	return order, nil
}

// attachSlot best-effort loads the order's slot for action responses.
func (s *TradeOrderService) attachSlot(ctx context.Context, order *model.TradeOrder) {
	if order.SlotID == nil {
		return
	}
	slot, err := s.slots.FindByID(ctx, *order.SlotID)
	if err != nil {
		s.logger.Warn(fmt.Sprintf("trade_order[id=%d] attach slot %d failed: %v", order.ID, *order.SlotID, err))
		return
	}
	order.Slot = slot
}
