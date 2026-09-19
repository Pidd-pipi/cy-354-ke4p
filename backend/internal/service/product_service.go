package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lp/campus-market/internal/constants"
	"github.com/lp/campus-market/internal/dto"
	"github.com/lp/campus-market/internal/model"
	"github.com/lp/campus-market/internal/util"
)

// ProductRepository is the data access contract for product rows.
type ProductRepository interface {
	Create(ctx context.Context, p *model.Product) error
	FindByID(ctx context.Context, id uint) (*model.Product, error)
	FindByIDForUpdate(ctx context.Context, id uint) (*model.Product, error)
	List(ctx context.Context, category, campus, keyword, status string, page, pageSize int) ([]model.Product, int64, error)
	UpdateStatus(ctx context.Context, id uint, status string) error
	Count(ctx context.Context) (int64, error)
}

// SlotStore is the data access contract for product meetup slots.
type SlotStore interface {
	CreateBatch(ctx context.Context, slots []model.ProductSlot) error
	ListByProduct(ctx context.Context, productID uint) ([]model.ProductSlot, error)
	FindByProductAndStartForUpdate(ctx context.Context, productID uint, startAt time.Time) (*model.ProductSlot, error)
	Occupy(ctx context.Context, slotID, orderID uint) error
	Release(ctx context.Context, slotID uint) error
	LockAllForProduct(ctx context.Context, productID, orderID uint) (int64, error)
}

// TxRunner executes cross-repository writes atomically.
type TxRunner interface {
	Transaction(ctx context.Context, fn func(txCtx context.Context) error) error
}

// ProductService manages second-hand product publishing and lifecycle.
type ProductService struct {
	products ProductRepository
	slots    SlotStore
	tx       TxRunner
	logger   *slog.Logger
}

// NewProductService wires the product service dependencies.
func NewProductService(products ProductRepository, slots SlotStore, tx TxRunner, logger *slog.Logger) *ProductService {
	return &ProductService{products: products, slots: slots, tx: tx, logger: logger}
}

// Create publishes a new product together with its optional meetup slots.
func (s *ProductService) Create(ctx context.Context, sellerID uint, req *dto.CreateProductRequest) (*model.Product, []model.ProductSlot, error) {
	if !constants.IsProductCategory(req.Category) {
		return nil, nil, util.NewAppError(400, constants.CodeValidation, "商品分类不合法", nil)
	}
	slotTimes, err := util.ValidateSlotTimes(req.SlotStarts, time.Now(), constants.SlotMaxAdvanceDays, constants.SlotMaxPerProduct)
	if err != nil {
		return nil, nil, util.NewAppError(400, constants.CodeValidation, constants.MsgSlotInvalid, nil)
	}
	p := &model.Product{
		SellerID: sellerID, Title: req.Title, Description: req.Description,
		Price: req.Price, Category: req.Category, Condition: req.Condition,
		Campus: req.Campus, TradeLocation: req.TradeLocation, Images: req.Images,
		Status: constants.ProductStatusOnSale,
	}
	created := make([]model.ProductSlot, 0, len(slotTimes))
	if txErr := s.tx.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.products.Create(txCtx, p); err != nil {
			return err
		}
		for _, t := range slotTimes {
			created = append(created, model.ProductSlot{
				ProductID: p.ID, StartAt: t, Status: constants.ProductSlotStatusOpen,
			})
		}
		if err := s.slots.CreateBatch(txCtx, created); err != nil {
			return err
		}
		return nil
	}); txErr != nil {
		s.logger.Error(fmt.Sprintf(constants.LogProductPublishFailed, sellerID, req.Title, txErr))
		return nil, nil, util.WrapAppError(fmt.Errorf("product[seller=%d] publish: %w", sellerID, txErr), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	s.logger.Info(fmt.Sprintf(constants.LogProductPublishSuccess, p.ID, p.Title))
	if len(created) > 0 {
		s.logger.Info(fmt.Sprintf(constants.LogProductSlotPublishSuccess, p.ID, len(created)))
	}
	return p, created, nil
}

// Get returns one product.
func (s *ProductService) Get(ctx context.Context, id uint) (*model.Product, error) {
	p, err := s.products.FindByID(ctx, id)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("product[id=%d] get: %w", id, err), 404, constants.CodeNotFound, constants.MsgNotFound)
	}
	return p, nil
}

// GetDetail returns a product plus its meetup slot availability for the
// product detail page.
func (s *ProductService) GetDetail(ctx context.Context, id uint) (*dto.ProductDetailResponse, error) {
	p, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	slots, err := s.slots.ListByProduct(ctx, id)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("product[id=%d] slots get: %w", id, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	now := time.Now()
	views := make([]dto.ProductSlotView, 0, len(slots))
	open := 0
	for _, slot := range slots {
		available := slot.Status == constants.ProductSlotStatusOpen && slot.StartAt.After(now)
		if available {
			open++
		}
		views = append(views, dto.ProductSlotView{
			ID: slot.ID, StartAt: slot.StartAt.Format(time.RFC3339),
			Status: slot.Status, Available: available,
		})
	}
	p.HasSlots = len(views) > 0
	p.OpenSlotCount = open
	return &dto.ProductDetailResponse{
		Product:    p,
		Slots:      views,
		TotalSlots: len(views),
		OpenSlots:  open,
		Bookable:   p.Status == constants.ProductStatusOnSale && (len(views) == 0 || open > 0),
	}, nil
}

// List filters products.
func (s *ProductService) List(ctx context.Context, q *dto.ListProductQuery) (*dto.PageResult, error) {
	q.Normalize()
	items, total, err := s.products.List(ctx, q.Category, q.Campus, q.Keyword, q.Status, q.Page, q.PageSize)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("product list: %w", err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	return &dto.PageResult{Items: items, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

// Remove lets the seller take down a product.
func (s *ProductService) Remove(ctx context.Context, sellerID, productID uint) (*model.Product, error) {
	p, err := s.products.FindByID(ctx, productID)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("product[id=%d] remove find: %w", productID, err), 404, constants.CodeNotFound, constants.MsgNotFound)
	}
	if p.SellerID != sellerID {
		return nil, util.NewAppError(403, constants.CodeForbidden, constants.MsgForbidden, nil)
	}
	if err := s.products.UpdateStatus(ctx, productID, constants.ProductStatusRemoved); err != nil {
		return nil, util.WrapAppError(fmt.Errorf("product[id=%d] remove: %w", productID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	s.logger.Info(fmt.Sprintf(constants.LogProductRemoveSuccess, productID))
	p.Status = constants.ProductStatusRemoved
	return p, nil
}

// MarkSold sets the product as sold after trade completion.
func (s *ProductService) MarkSold(ctx context.Context, productID uint) error {
	if err := s.products.UpdateStatus(ctx, productID, constants.ProductStatusSold); err != nil {
		return util.WrapAppError(fmt.Errorf("product[id=%d] mark sold: %w", productID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	s.logger.Info(fmt.Sprintf(constants.LogProductSoldSuccess, productID))
	return nil
}
