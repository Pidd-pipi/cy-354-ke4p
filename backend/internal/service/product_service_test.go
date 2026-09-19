package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/lp/campus-market/internal/constants"
	"github.com/lp/campus-market/internal/dto"
	"github.com/lp/campus-market/internal/model"
	"github.com/lp/campus-market/internal/util"
)

type fakeProductRepo struct {
	products map[uint]*model.Product
	nextID   uint
}

func newFakeProductRepo() *fakeProductRepo {
	return &fakeProductRepo{products: map[uint]*model.Product{}, nextID: 1}
}

func (f *fakeProductRepo) Create(_ context.Context, p *model.Product) error {
	p.ID = f.nextID
	f.nextID++
	f.products[p.ID] = p
	return nil
}

func (f *fakeProductRepo) FindByID(_ context.Context, id uint) (*model.Product, error) {
	if p, ok := f.products[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, util.ErrNotFound
}

func (f *fakeProductRepo) FindByIDForUpdate(ctx context.Context, id uint) (*model.Product, error) {
	return f.FindByID(ctx, id)
}

func (f *fakeProductRepo) List(_ context.Context, category, campus, keyword, status string, page, pageSize int) ([]model.Product, int64, error) {
	var out []model.Product
	for _, p := range f.products {
		if category != "" && p.Category != category {
			continue
		}
		if campus != "" && p.Campus != campus {
			continue
		}
		if status != "" && p.Status != status {
			continue
		}
		out = append(out, *p)
	}
	return out, int64(len(out)), nil
}

func (f *fakeProductRepo) UpdateStatus(_ context.Context, id uint, status string) error {
	_, err := f.FindByID(context.Background(), id)
	if err != nil {
		return err
	}
	f.products[id].Status = status
	return nil
}

func (f *fakeProductRepo) Count(context.Context) (int64, error) { return int64(len(f.products)), nil }

func TestProductServiceCreate(t *testing.T) {
	svc := NewProductService(newFakeProductRepo(), newFakeSlotStore(), fakeTxRunner{}, slog.Default())
	tests := []struct {
		name     string
		category string
		wantErr  bool
	}{
		{name: "valid books", category: constants.ProductCategoryBooks, wantErr: false},
		{name: "valid electronics", category: constants.ProductCategoryElectronics, wantErr: false},
		{name: "invalid category", category: "sports", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &dto.CreateProductRequest{Title: "测试商品", Price: 10, Category: tt.category, Condition: "全新", Campus: "东校区", TradeLocation: "东门"}
			_, _, err := svc.Create(context.Background(), 1, req)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestProductServiceCreateSlots(t *testing.T) {
	repo := newFakeProductRepo()
	slots := newFakeSlotStore()
	svc := NewProductService(repo, slots, fakeTxRunner{}, slog.Default())
	future := time.Now().Add(24 * time.Hour)
	future = time.Date(future.Year(), future.Month(), future.Day(), future.Hour(), 30, 0, 0, future.Location())
	req := &dto.CreateProductRequest{
		Title: "面交商品", Price: 10, Category: constants.ProductCategoryBooks, Condition: "全新",
		Campus: "东校区", TradeLocation: "东门",
		SlotStarts: []string{future.Format(time.RFC3339)},
	}
	p, created, err := svc.Create(context.Background(), 1, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("expected 1 slot, got %d", len(created))
	}
	detail, err := svc.GetDetail(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.TotalSlots != 1 || detail.OpenSlots != 1 || !detail.Bookable {
		t.Fatalf("unexpected detail: %+v", detail)
	}
}

func TestProductServiceCreateSlotRejectsUnaligned(t *testing.T) {
	svc := NewProductService(newFakeProductRepo(), newFakeSlotStore(), fakeTxRunner{}, slog.Default())
	bad := time.Now().Add(24 * time.Hour).Format(time.RFC3339) // minutes not aligned
	req := &dto.CreateProductRequest{
		Title: "面交商品", Price: 10, Category: constants.ProductCategoryBooks, Condition: "全新",
		Campus: "东校区", TradeLocation: "东门", SlotStarts: []string{bad},
	}
	if _, _, err := svc.Create(context.Background(), 1, req); err == nil {
		t.Fatalf("expected validation error for unaligned slot")
	}
}

func TestProductServiceRemoveOwnership(t *testing.T) {
	repo := newFakeProductRepo()
	svc := NewProductService(repo, newFakeSlotStore(), fakeTxRunner{}, slog.Default())
	created, _, _ := svc.Create(context.Background(), 1, &dto.CreateProductRequest{Title: "我的书", Price: 10, Category: constants.ProductCategoryBooks, Condition: "全新", Campus: "东校区", TradeLocation: "东门"})
	if _, err := svc.Remove(context.Background(), 99, created.ID); err == nil {
		t.Fatalf("expected forbidden error for non-owner")
	}
	removed, err := svc.Remove(context.Background(), 1, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed.Status != constants.ProductStatusRemoved {
		t.Fatalf("expected removed status")
	}
}
