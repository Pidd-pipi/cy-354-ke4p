package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/lp/campus-market/internal/constants"
	"github.com/lp/campus-market/internal/dto"
)

func newTestProductService(db *fakeDB) (*ProductService, *fakeProductRepo, *fakeSlotRepo) {
	products := newFakeProductRepo(db)
	slots := newFakeSlotRepo(db)
	slotSvc := NewTradeSlotService(slots, slog.Default())
	svc := NewProductService(products, slots, slotSvc, slog.Default())
	return svc, products, slots
}

func TestProductServiceCreate(t *testing.T) {
	svc, _, _ := newTestProductService(newFakeDB())
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
			_, err := svc.Create(context.Background(), 1, req)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestProductServiceRemoveOwnership(t *testing.T) {
	db := newFakeDB()
	svc, _, _ := newTestProductService(db)
	created, _ := svc.Create(context.Background(), 1, &dto.CreateProductRequest{Title: "我的书", Price: 10, Category: constants.ProductCategoryBooks, Condition: "全新", Campus: "东校区", TradeLocation: "东门"})
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

func TestProductServiceCreateWithSlots(t *testing.T) {
	db := newFakeDB()
	svc, _, slotRepo := newTestProductService(db)
	future := time.Now().Add(3 * time.Hour).Truncate(time.Hour)
	req := &dto.CreateProductRequest{
		Title: "带时段的商品", Price: 10, Category: constants.ProductCategoryBooks,
		Condition: "全新", Campus: "东校区", TradeLocation: "东门",
		Slots: []dto.CreateSlotInput{{StartTime: future}, {StartTime: future.Add(30 * time.Minute)}},
	}
	p, err := svc.Create(context.Background(), 1, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	slots, _ := slotRepo.ListByProduct(context.Background(), p.ID)
	if len(slots) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(slots))
	}
	if slots[0].EndTime.Sub(slots[0].StartTime) != constants.SlotDuration {
		t.Fatalf("slot must last 30 minutes")
	}
}

func TestProductServiceCreateRejectsBadSlots(t *testing.T) {
	svc, _, _ := newTestProductService(newFakeDB())
	base := &dto.CreateProductRequest{Title: "坏时段", Price: 10, Category: constants.ProductCategoryBooks, Condition: "全新", Campus: "东校区", TradeLocation: "东门"}

	t.Run("expired slot", func(t *testing.T) {
		req := *base
		req.Slots = []dto.CreateSlotInput{{StartTime: time.Now().Add(-time.Hour)}}
		if _, err := svc.Create(context.Background(), 1, &req); err == nil {
			t.Fatalf("expected error for past slot")
		}
	})
	t.Run("off-grid slot", func(t *testing.T) {
		req := *base
		req.Slots = []dto.CreateSlotInput{{StartTime: time.Now().Add(3 * time.Hour).Truncate(time.Hour).Add(17 * time.Minute)}}
		if _, err := svc.Create(context.Background(), 1, &req); err == nil {
			t.Fatalf("expected error for non half-hour slot")
		}
	})
	t.Run("duplicate slot", func(t *testing.T) {
		req := *base
		at := time.Now().Add(3 * time.Hour).Truncate(time.Hour)
		req.Slots = []dto.CreateSlotInput{{StartTime: at}, {StartTime: at}}
		if _, err := svc.Create(context.Background(), 1, &req); err == nil {
			t.Fatalf("expected error for duplicate slot")
		}
	})
}
