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

// orderStore is the data access contract for trade order rows.
type orderStore interface {
	Create(ctx context.Context, o *model.TradeOrder) error
	FindByID(ctx context.Context, id uint) (*model.TradeOrder, error)
	FindByProductAndBuyer(ctx context.Context, productID, buyerID uint) (*model.TradeOrder, error)
	ListByUser(ctx context.Context, userID uint, page, pageSize int) ([]model.TradeOrder, int64, error)
	UpdateStatus(ctx context.Context, id uint, status string) error
	UpdateStatusFromPending(ctx context.Context, id uint, status string) error
	UpdateBuyerConfirmed(ctx context.Context, id uint, ts interface{}) error
	UpdateSellerConfirmed(ctx context.Context, id uint, ts interface{}) error
	Transaction(ctx context.Context, fn func(txCtx context.Context) error) error
}

// productForOrderStore is the product data access needed by trade orders.
type productForOrderStore interface {
	FindByIDForUpdate(ctx context.Context, id uint) (*model.Product, error)
	UpdateStatus(ctx context.Context, id uint, status string) error
}

// TradeOrderService manages purchase intents, confirmations and completion.
type TradeOrderService struct {
	orders   orderStore
	products productForOrderStore
	slots    SlotStore
	logger   *slog.Logger
}

// NewTradeOrderService wires the trade order service dependencies.
func NewTradeOrderService(orders orderStore, products productForOrderStore, slots SlotStore, logger *slog.Logger) *TradeOrderService {
	return &TradeOrderService{orders: orders, products: products, slots: slots, logger: logger}
}

// Create creates a pending trade order. For products that publish meetup
// slots the buyer must pick a non-expired, open half-hour slot. The product
// row and the slot row are locked inside one transaction, so concurrent
// contenders for the same slot serialize: the loser gets a slot-unavailable
// conflict and never leaves a duplicate order.
func (s *TradeOrderService) Create(ctx context.Context, buyer *model.User, req *dto.CreateTradeOrderRequest) (*model.TradeOrder, error) {
	var order *model.TradeOrder
	err := s.orders.Transaction(ctx, func(txCtx context.Context) error {
		product, err := s.products.FindByIDForUpdate(txCtx, req.ProductID)
		if err != nil {
			if errors.Is(err, util.ErrNotFound) {
				return util.NewAppError(404, constants.CodeNotFound, constants.MsgNotFound, nil)
			}
			return fmt.Errorf("trade_order[buyer=%d] product lookup: %w", buyer.ID, err)
		}
		if product.SellerID == buyer.ID {
			return util.NewAppError(400, constants.CodeBadRequest, "不能购买自己的商品", nil)
		}
		if product.Status != constants.ProductStatusOnSale {
			return util.NewAppError(409, constants.CodeConflict, constants.MsgProductNotOnSale, nil)
		}

		slots, err := s.slots.ListByProduct(txCtx, req.ProductID)
		if err != nil {
			return fmt.Errorf("trade_order[buyer=%d] slot list: %w", buyer.ID, err)
		}

		order = &model.TradeOrder{
			ProductID: req.ProductID, BuyerID: buyer.ID, SellerID: product.SellerID,
			Status: constants.TradeStatusPending,
		}

		if len(slots) > 0 {
			// Slot-based face-to-face booking path.
			if req.SlotStart == "" {
				return util.NewAppError(400, constants.CodeBadRequest, constants.MsgSlotRequired, nil)
			}
			startAt, perr := util.ParseSlotStart(req.SlotStart)
			if perr != nil || !util.IsHalfHourAligned(startAt) {
				return util.NewAppError(400, constants.CodeBadRequest, constants.MsgSlotInvalid, nil)
			}
			if !startAt.After(time.Now()) {
				return util.NewAppError(409, constants.CodeSlotUnavailable, constants.MsgSlotExpired, nil)
			}
			slotStart := startAt
			order.SlotStartAt = &slotStart
			slot, ferr := s.slots.FindByProductAndStartForUpdate(txCtx, req.ProductID, startAt)
			if ferr != nil {
				if errors.Is(ferr, util.ErrNotFound) {
					return util.NewAppError(409, constants.CodeSlotUnavailable, constants.MsgSlotUnavailable, nil)
				}
				return fmt.Errorf("trade_order[buyer=%d] slot lookup: %w", buyer.ID, ferr)
			}
			if slot.Status != constants.ProductSlotStatusOpen {
				return util.NewAppError(409, constants.CodeSlotUnavailable, constants.MsgSlotUnavailable, nil)
			}
			if !slot.StartAt.After(time.Now()) {
				return util.NewAppError(409, constants.CodeSlotUnavailable, constants.MsgSlotExpired, nil)
			}
			if err := s.orders.Create(txCtx, order); err != nil {
				return fmt.Errorf("trade_order[buyer=%d] create: %w", buyer.ID, err)
			}
			if err := s.slots.Occupy(txCtx, slot.ID, order.ID); err != nil {
				if errors.Is(err, util.ErrConflict) {
					return util.NewAppError(409, constants.CodeSlotUnavailable, constants.MsgSlotUnavailable, nil)
				}
				s.logger.Error(fmt.Sprintf(constants.LogProductSlotOccupyFailed, slot.ID, req.ProductID, order.ID, err))
				return fmt.Errorf("trade_order[buyer=%d] slot occupy: %w", buyer.ID, err)
			}
			if err := s.products.UpdateStatus(txCtx, req.ProductID, constants.ProductStatusReserved); err != nil {
				return fmt.Errorf("trade_order[buyer=%d] product reserve: %w", buyer.ID, err)
			}
			s.logger.Info(fmt.Sprintf(constants.LogProductSlotOccupySuccess, slot.ID, req.ProductID, order.ID))
			return nil
		}

		// Legacy direct-order path for products without meetup slots.
		if existing, err := s.orders.FindByProductAndBuyer(txCtx, req.ProductID, buyer.ID); err == nil && existing != nil {
			return util.NewAppError(409, constants.CodeConflict, "您已对该商品下单", nil)
		}
		if err := s.orders.Create(txCtx, order); err != nil {
			return fmt.Errorf("trade_order[buyer=%d] create: %w", buyer.ID, err)
		}
		return nil
	})
	if err != nil {
		// Business AppErrors are returned as-is so the handler keeps the
		// {entity, field, role} message produced above.
		var appErr *util.AppError
		if errors.As(err, &appErr) {
			return nil, appErr
		}
		return nil, util.WrapAppError(fmt.Errorf("trade_order[buyer=%d] create: %w", buyer.ID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	s.logger.Info(fmt.Sprintf(constants.LogTradeOrderCreateSuccess, order.ID, req.ProductID))
	return order, nil
}

// ListMy returns the orders where the user participates.
func (s *TradeOrderService) ListMy(ctx context.Context, userID uint, q *dto.PageQuery) (*dto.PageResult, error) {
	q.Normalize()
	items, total, err := s.orders.ListByUser(ctx, userID, q.Page, q.PageSize)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("trade_order[user=%d] list: %w", userID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	return &dto.PageResult{Items: items, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

// BuyerConfirm marks the order confirmed by the buyer.
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
	return order, nil
}

// SellerConfirm completes the order, marks the product sold and permanently
// locks all its meetup slots.
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
	var locked int64
	if err := s.orders.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.orders.UpdateSellerConfirmed(txCtx, orderID, now); err != nil {
			return err
		}
		if err := s.products.UpdateStatus(txCtx, order.ProductID, constants.ProductStatusSold); err != nil {
			return err
		}
		n, err := s.slots.LockAllForProduct(txCtx, order.ProductID, orderID)
		if err != nil {
			return err
		}
		locked = n
		return nil
	}); err != nil {
		if errors.Is(err, util.ErrConflict) {
			return nil, util.NewAppError(409, constants.CodeConflict, constants.MsgTradeStatusInvalid, nil)
		}
		s.logger.Error(fmt.Sprintf(constants.LogTradeOrderCompleteFailed, orderID, err))
		return nil, util.WrapAppError(fmt.Errorf("trade_order[id=%d] seller confirm: %w", orderID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	s.logger.Info(fmt.Sprintf(constants.LogTradeOrderCompleteSuccess, orderID, order.ProductID))
	if locked > 0 {
		s.logger.Info(fmt.Sprintf(constants.LogProductSlotLockedSuccess, order.ProductID, locked))
	}
	order.Status = constants.TradeStatusCompleted
	return order, nil
}

// Cancel cancels a pending order. Either party may cancel during the pending
// stage; for slot bookings the slot is released and the product goes back on
// sale so other buyers can book it.
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

	if order.SlotStartAt != nil {
		if err := s.cancelSlotOrder(ctx, order); err != nil {
			return nil, err
		}
	} else {
		if err := s.orders.UpdateStatusFromPending(ctx, orderID, constants.TradeStatusCancelled); err != nil {
			if errors.Is(err, util.ErrConflict) {
				return nil, util.NewAppError(409, constants.CodeConflict, constants.MsgTradeStatusInvalid, nil)
			}
			return nil, util.WrapAppError(fmt.Errorf("trade_order[id=%d] cancel: %w", orderID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
		}
	}
	s.logger.Info(fmt.Sprintf(constants.LogTradeOrderCancelSuccess, orderID))
	order.Status = constants.TradeStatusCancelled
	return order, nil
}

// cancelSlotOrder atomically cancels a slot booking, frees the slot and
// reopens the product.
func (s *TradeOrderService) cancelSlotOrder(ctx context.Context, order *model.TradeOrder) error {
	return s.orders.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.orders.UpdateStatusFromPending(txCtx, order.ID, constants.TradeStatusCancelled); err != nil {
			if errors.Is(err, util.ErrConflict) {
				return util.NewAppError(409, constants.CodeConflict, constants.MsgTradeStatusInvalid, nil)
			}
			return fmt.Errorf("trade_order[id=%d] cancel slot order: %w", order.ID, err)
		}
		slot, err := s.slots.FindByProductAndStartForUpdate(txCtx, order.ProductID, *order.SlotStartAt)
		if err != nil {
			return fmt.Errorf("trade_order[id=%d] cancel slot lookup: %w", order.ID, err)
		}
		if err := s.slots.Release(txCtx, slot.ID); err != nil {
			return fmt.Errorf("trade_order[id=%d] slot release: %w", order.ID, err)
		}
		if err := s.products.UpdateStatus(txCtx, order.ProductID, constants.ProductStatusOnSale); err != nil {
			return fmt.Errorf("trade_order[id=%d] product reopen: %w", order.ID, err)
		}
		s.logger.Info(fmt.Sprintf(constants.LogProductSlotReleasedSuccess, slot.ID, order.ProductID, order.ID))
		return nil
	})
}
