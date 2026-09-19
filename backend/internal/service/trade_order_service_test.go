package service

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/lp/campus-market/internal/constants"
	"github.com/lp/campus-market/internal/dto"
	"github.com/lp/campus-market/internal/model"
)

// seedProductWithSlot creates a seller, a product and one future half-hour
// slot, returning the wired service plus the created identifiers.
func seedProductWithSlot(t *testing.T) (svc *TradeOrderService, db *fakeDB, productID, slotID uint) {
	t.Helper()
	db = newFakeDB()
	products := newFakeProductRepo(db)
	orders := newFakeOrderRepo(db)
	slots := newFakeSlotRepo(db)
	svc = NewTradeOrderService(orders, products, slots, slog.Default())

	productID = 1
	db.products[productID] = &model.Product{ID: productID, SellerID: 1, Status: constants.ProductStatusOnSale}
	future := time.Now().Add(24 * time.Hour).Truncate(time.Hour)
	if err := slots.CreateBatch(context.Background(), []model.TradeSlot{{
		ProductID: productID, StartTime: future, EndTime: future.Add(constants.SlotDuration),
		Status: constants.SlotStatusAvailable,
	}}); err != nil {
		t.Fatalf("seed slot: %v", err)
	}
	list, _ := slots.ListByProduct(context.Background(), productID)
	slotID = list[0].ID
	return svc, db, productID, slotID
}

func TestTradeOrderCreateRequiresSlot(t *testing.T) {
	svc, db, productID, _ := seedProductWithSlot(t)
	buyer := &model.User{ID: 2}
	if _, err := svc.Create(context.Background(), buyer, &dto.CreateTradeOrderRequest{ProductID: productID}); err == nil {
		t.Fatalf("expected error when slot is required")
	}
	if len(db.orders) != 0 {
		t.Fatalf("failed order creation must not leave an order row")
	}
}

func TestTradeOrderSlotClosedLoop(t *testing.T) {
	svc, db, productID, slotID := seedProductWithSlot(t)
	buyerA := &model.User{ID: 2}
	buyerB := &model.User{ID: 3}

	// First buyer grabs the only slot successfully.
	order, err := svc.Create(context.Background(), buyerA, &dto.CreateTradeOrderRequest{ProductID: productID, SlotID: &slotID})
	if err != nil {
		t.Fatalf("first buyer create: %v", err)
	}
	if got := db.slots[slotID].Status; got != constants.SlotStatusLocked {
		t.Fatalf("slot should be locked, got %s", got)
	}

	// Second buyer racing for the same slot must fail and leave no order.
	orderCountBefore := len(db.orders)
	if _, err := svc.Create(context.Background(), buyerB, &dto.CreateTradeOrderRequest{ProductID: productID, SlotID: &slotID}); err == nil {
		t.Fatalf("second buyer must not grab the same slot")
	}
	if len(db.orders) != orderCountBefore {
		t.Fatalf("a duplicate order must not be created on slot conflict")
	}

	// Either party cancels while pending: the slot is released and reusable.
	if _, err := svc.Cancel(context.Background(), buyerA.ID, order.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if got := db.slots[slotID].Status; got != constants.SlotStatusReleased {
		t.Fatalf("slot should be released, got %s", got)
	}
	if db.slots[slotID].OrderID != nil {
		t.Fatalf("released slot must detach from the order")
	}

	// The second buyer can now book the released slot.
	order2, err := svc.Create(context.Background(), buyerB, &dto.CreateTradeOrderRequest{ProductID: productID, SlotID: &slotID})
	if err != nil {
		t.Fatalf("second buyer create after release: %v", err)
	}
	if got := db.slots[slotID].Status; got != constants.SlotStatusLocked {
		t.Fatalf("slot should be locked again, got %s", got)
	}

	// Buyer confirms then seller confirms (completion): slot is permanently locked.
	if _, err := svc.BuyerConfirm(context.Background(), buyerB.ID, order2.ID); err != nil {
		t.Fatalf("buyer confirm: %v", err)
	}
	if _, err := svc.SellerConfirm(context.Background(), 1, order2.ID); err != nil {
		t.Fatalf("seller confirm: %v", err)
	}
	if got := db.slots[slotID].Status; got != constants.SlotStatusLocked {
		t.Fatalf("completed order slot must stay locked, got %s", got)
	}

	// Cancellation after completion must be rejected and keep the lock.
	if _, err := svc.Cancel(context.Background(), buyerB.ID, order2.ID); err == nil {
		t.Fatalf("completed order must not be cancellable")
	}
	if got := db.slots[slotID].Status; got != constants.SlotStatusLocked {
		t.Fatalf("slot lock must survive rejected cancellation, got %s", got)
	}
}

func TestTradeOrderSellerCanCancelPending(t *testing.T) {
	svc, db, productID, slotID := seedProductWithSlot(t)
	buyer := &model.User{ID: 2}
	order, err := svc.Create(context.Background(), buyer, &dto.CreateTradeOrderRequest{ProductID: productID, SlotID: &slotID})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// The seller (user 1) cancels while still pending: slot released.
	if _, err := svc.Cancel(context.Background(), 1, order.ID); err != nil {
		t.Fatalf("seller cancel pending order: %v", err)
	}
	if got := db.slots[slotID].Status; got != constants.SlotStatusReleased {
		t.Fatalf("seller cancellation should release the slot, got %s", got)
	}
}

func TestTradeOrderConcurrentGrabOnlyOneWins(t *testing.T) {
	svc, db, productID, slotID := seedProductWithSlot(t)

	const contenders = 30
	var wg sync.WaitGroup
	wg.Add(contenders)
	wins := make(chan uint, contenders)
	start := make(chan struct{})
	for i := 0; i < contenders; i++ {
		buyerID := uint(100 + i)
		go func() {
			defer wg.Done()
			<-start
			o, err := svc.Create(context.Background(), &model.User{ID: buyerID},
				&dto.CreateTradeOrderRequest{ProductID: productID, SlotID: &slotID})
			if err == nil {
				wins <- o.ID
			}
		}()
	}
	close(start)
	wg.Wait()
	close(wins)

	var winners []uint
	for id := range wins {
		winners = append(winners, id)
	}
	if len(winners) != 1 {
		t.Fatalf("expected exactly one winner for the slot, got %d", len(winners))
	}
	// Only the winning pending order should remain tied to the product.
	var liveOrders int
	for _, o := range db.orders {
		if o.ProductID == productID && o.Status == constants.TradeStatusPending {
			liveOrders++
		}
	}
	if liveOrders != 1 {
		t.Fatalf("expected one live order, got %d", liveOrders)
	}
}

func TestTradeOrderExpiredSlotRejected(t *testing.T) {
	db := newFakeDB()
	products := newFakeProductRepo(db)
	orders := newFakeOrderRepo(db)
	slots := newFakeSlotRepo(db)
	svc := NewTradeOrderService(orders, products, slots, slog.Default())

	db.products[1] = &model.Product{ID: 1, SellerID: 10, Status: constants.ProductStatusOnSale}
	past := time.Now().Add(-time.Hour).Truncate(time.Hour)
	if err := slots.CreateBatch(context.Background(), []model.TradeSlot{{
		ProductID: 1, StartTime: past, EndTime: past.Add(constants.SlotDuration),
		Status: constants.SlotStatusAvailable,
	}}); err != nil {
		t.Fatalf("seed slot: %v", err)
	}
	list, _ := slots.ListByProduct(context.Background(), 1)
	buyer := &model.User{ID: 2}
	if _, err := svc.Create(context.Background(), buyer, &dto.CreateTradeOrderRequest{ProductID: 1, SlotID: &list[0].ID}); err == nil {
		t.Fatalf("expired slot must be rejected")
	}
}

func TestTradeOrderWithoutSlotStillWorks(t *testing.T) {
	db := newFakeDB()
	products := newFakeProductRepo(db)
	orders := newFakeOrderRepo(db)
	slots := newFakeSlotRepo(db)
	svc := NewTradeOrderService(orders, products, slots, slog.Default())

	db.products[1] = &model.Product{ID: 1, SellerID: 10, Status: constants.ProductStatusOnSale}
	buyer := &model.User{ID: 2}
	order, err := svc.Create(context.Background(), buyer, &dto.CreateTradeOrderRequest{ProductID: 1})
	if err != nil {
		t.Fatalf("slot-free product order should succeed: %v", err)
	}
	if order.SlotID != nil {
		t.Fatalf("slot-free order should not carry a slot")
	}
}
