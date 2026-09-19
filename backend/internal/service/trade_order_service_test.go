package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/lp/campus-market/internal/constants"
	"github.com/lp/campus-market/internal/dto"
	"github.com/lp/campus-market/internal/model"
)

func newSlotBookingFixture(t *testing.T) (*TradeOrderService, *fakeProductRepo, *fakeSlotStore, *fakeOrderStore, time.Time) {
	t.Helper()
	products := newFakeProductRepo()
	slots := newFakeSlotStore()
	orders := newFakeOrderStore()
	svc := NewTradeOrderService(orders, products, slots, slog.Default())

	product := &model.Product{ID: 1, SellerID: 10, Status: constants.ProductStatusOnSale}
	products.products[1] = product
	future := time.Now().Add(24 * time.Hour)
	future = time.Date(future.Year(), future.Month(), future.Day(), future.Hour(), 30, 0, 0, future.Location())
	slot := &model.ProductSlot{ID: 100, ProductID: 1, StartAt: future, Status: constants.ProductSlotStatusOpen}
	slots.slots[100] = slot
	return svc, products, slots, orders, future
}

func TestTradeOrderCreateBooksSlot(t *testing.T) {
	svc, products, slots, orders, future := newSlotBookingFixture(t)
	buyer := &model.User{ID: 20}
	order, err := svc.Create(context.Background(), buyer, &dto.CreateTradeOrderRequest{
		ProductID: 1, SlotStart: future.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.SlotStartAt == nil || !order.SlotStartAt.Equal(future) {
		t.Fatalf("expected slot start on order, got %+v", order.SlotStartAt)
	}
	if slots.slots[100].Status != constants.ProductSlotStatusOccupied {
		t.Fatalf("expected slot occupied, got %s", slots.slots[100].Status)
	}
	if products.products[1].Status != constants.ProductStatusReserved {
		t.Fatalf("expected product reserved, got %s", products.products[1].Status)
	}
	if len(orders.created) != 1 {
		t.Fatalf("expected exactly one order, got %d", len(orders.created))
	}
}

func TestTradeOrderCreateSlotRaceLeavesSingleOrder(t *testing.T) {
	svc, _, _, orders, future := newSlotBookingFixture(t)

	// First buyer wins the slot.
	if _, err := svc.Create(context.Background(), &model.User{ID: 20}, &dto.CreateTradeOrderRequest{
		ProductID: 1, SlotStart: future.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("first booking failed: %v", err)
	}
	// Second buyer races for the same occupied slot and must fail.
	if _, err := svc.Create(context.Background(), &model.User{ID: 21}, &dto.CreateTradeOrderRequest{
		ProductID: 1, SlotStart: future.Format(time.RFC3339),
	}); err == nil {
		t.Fatalf("expected slot conflict for second buyer")
	}
	if len(orders.created) != 1 {
		t.Fatalf("expected a single order after the race, got %d: %v", len(orders.created), orders.created)
	}
}

func TestTradeOrderCreateRequiresSlot(t *testing.T) {
	svc, _, _, _, future := newSlotBookingFixture(t)
	if _, err := svc.Create(context.Background(), &model.User{ID: 20}, &dto.CreateTradeOrderRequest{ProductID: 1}); err == nil {
		t.Fatalf("expected error when slot_start is missing")
	}
	// Expired slot is rejected.
	expired := time.Now().Add(-time.Hour)
	if _, err := svc.Create(context.Background(), &model.User{ID: 20}, &dto.CreateTradeOrderRequest{
		ProductID: 1, SlotStart: expired.Format(time.RFC3339),
	}); err == nil {
		t.Fatalf("expected error for expired slot")
	}
	_ = future
}

func TestTradeOrderCancelReleasesSlotAndReopensProduct(t *testing.T) {
	svc, products, slots, _, future := newSlotBookingFixture(t)
	order, err := svc.Create(context.Background(), &model.User{ID: 20}, &dto.CreateTradeOrderRequest{
		ProductID: 1, SlotStart: future.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("booking failed: %v", err)
	}
	cancelled, err := svc.Cancel(context.Background(), 20, order.ID)
	if err != nil {
		t.Fatalf("cancel failed: %v", err)
	}
	if cancelled.Status != constants.TradeStatusCancelled {
		t.Fatalf("expected cancelled, got %s", cancelled.Status)
	}
	if slots.slots[100].Status != constants.ProductSlotStatusOpen {
		t.Fatalf("expected slot released, got %s", slots.slots[100].Status)
	}
	if products.products[1].Status != constants.ProductStatusOnSale {
		t.Fatalf("expected product reopened, got %s", products.products[1].Status)
	}
	// Slot is bookable again by another buyer.
	if _, err := svc.Create(context.Background(), &model.User{ID: 21}, &dto.CreateTradeOrderRequest{
		ProductID: 1, SlotStart: future.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("rebooking after release failed: %v", err)
	}
}

func TestTradeOrderCompleteLocksSlots(t *testing.T) {
	svc, products, slots, _, future := newSlotBookingFixture(t)
	order, err := svc.Create(context.Background(), &model.User{ID: 20}, &dto.CreateTradeOrderRequest{
		ProductID: 1, SlotStart: future.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("booking failed: %v", err)
	}
	if _, err := svc.BuyerConfirm(context.Background(), 20, order.ID); err != nil {
		t.Fatalf("buyer confirm failed: %v", err)
	}
	if _, err := svc.SellerConfirm(context.Background(), 10, order.ID); err != nil {
		t.Fatalf("seller confirm failed: %v", err)
	}
	if products.products[1].Status != constants.ProductStatusSold {
		t.Fatalf("expected product sold, got %s", products.products[1].Status)
	}
	if slots.slots[100].Status != constants.ProductSlotStatusLocked {
		t.Fatalf("expected slot locked, got %s", slots.slots[100].Status)
	}
}

func TestTradeOrderLegacyPathWithoutSlots(t *testing.T) {
	products := newFakeProductRepo()
	slots := newFakeSlotStore()
	orders := newFakeOrderStore()
	svc := NewTradeOrderService(orders, products, slots, slog.Default())
	products.products[1] = &model.Product{ID: 1, SellerID: 10, Status: constants.ProductStatusOnSale}

	if _, err := svc.Create(context.Background(), &model.User{ID: 20}, &dto.CreateTradeOrderRequest{ProductID: 1}); err != nil {
		t.Fatalf("legacy order failed: %v", err)
	}
	if products.products[1].Status != constants.ProductStatusOnSale {
		t.Fatalf("legacy path must not change product status, got %s", products.products[1].Status)
	}
}
