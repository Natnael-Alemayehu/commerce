package service

import (
	"context"
	"errors"
	"fmt"

	"starterkit/internal/store"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// InventoryService handles inventory business logic.
type InventoryService struct {
	store *store.Store
}

// NewInventoryService creates a new InventoryService.
func NewInventoryService(store *store.Store) *InventoryService {
	return &InventoryService{store: store}
}

// ErrInsufficientStock is returned when there isn't enough available stock.
var ErrInsufficientStock = errors.New("insufficient stock")

// ErrInventoryNotFound is returned when an inventory record is not found.
var ErrInventoryNotFound = errors.New("inventory not found")

// AvailableStock returns the available (non-reserved) stock for a variant.
func (s *InventoryService) AvailableStock(ctx context.Context, variantID uuid.UUID) (int32, error) {
	inv, err := s.store.GetInventoryByVariant(ctx, variantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil // No inventory record means zero stock
		}
		return 0, fmt.Errorf("get inventory: %w", err)
	}
	return inv.Quantity - inv.ReservedQuantity, nil
}

// CheckStock verifies if the requested quantity is available.
func (s *InventoryService) CheckStock(ctx context.Context, variantID uuid.UUID, requestedQty int32) error {
	available, err := s.AvailableStock(ctx, variantID)
	if err != nil {
		return err
	}
	if available < requestedQty {
		return fmt.Errorf("%w: requested %d, available %d", ErrInsufficientStock, requestedQty, available)
	}
	return nil
}

// ReserveStock reserves inventory for an order.
func (s *InventoryService) ReserveStock(ctx context.Context, variantID uuid.UUID, quantity int32, orderID uuid.UUID) error {
	if quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}

	updated, err := s.store.ReserveStock(ctx, store.ReserveStockParams{
		VariantID:        variantID,
		ReservedQuantity: quantity,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: variant %s", ErrInsufficientStock, variantID)
		}
		return fmt.Errorf("reserve stock: %w", err)
	}

	// Verify the reservation actually happened
	if updated.ReservedQuantity < quantity {
		return fmt.Errorf("%w: variant %s", ErrInsufficientStock, variantID)
	}

	// Log movement
	if _, err := s.store.CreateStockMovement(ctx, store.CreateStockMovementParams{
		VariantID:    variantID,
		MovementType: "reservation",
		Quantity:     quantity,
		Reason:       pgtype.Text{String: "Order reservation", Valid: true},
		ReferenceID:  pgtype.UUID{Bytes: orderID, Valid: true},
	}); err != nil {
		return fmt.Errorf("log reservation movement: %w", err)
	}

	return nil
}

// ReleaseStock releases reserved inventory (e.g. on order cancellation).
func (s *InventoryService) ReleaseStock(ctx context.Context, variantID uuid.UUID, quantity int32, orderID uuid.UUID) error {
	if quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}

	_, err := s.store.ReleaseStock(ctx, store.ReleaseStockParams{
		VariantID:        variantID,
		ReservedQuantity: quantity,
	})
	if err != nil {
		return fmt.Errorf("release stock: %w", err)
	}

	// Log movement
	if _, err := s.store.CreateStockMovement(ctx, store.CreateStockMovementParams{
		VariantID:    variantID,
		MovementType: "release",
		Quantity:     quantity,
		Reason:       pgtype.Text{String: "Order cancellation", Valid: true},
		ReferenceID:  pgtype.UUID{Bytes: orderID, Valid: true},
	}); err != nil {
		return fmt.Errorf("log release movement: %w", err)
	}

	return nil
}

// ConfirmStockDeduction deducts reserved stock upon order confirmation/shipment.
func (s *InventoryService) ConfirmStockDeduction(ctx context.Context, variantID uuid.UUID, quantity int32, orderID uuid.UUID) error {
	if quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}

	_, err := s.store.ConfirmStockDeduction(ctx, store.ConfirmStockDeductionParams{
		VariantID: variantID,
		Quantity:  quantity,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: variant %s", ErrInsufficientStock, variantID)
		}
		return fmt.Errorf("confirm stock deduction: %w", err)
	}

	// Log movement
	if _, err := s.store.CreateStockMovement(ctx, store.CreateStockMovementParams{
		VariantID:    variantID,
		MovementType: "out",
		Quantity:     quantity,
		Reason:       pgtype.Text{String: "Order fulfillment", Valid: true},
		ReferenceID:  pgtype.UUID{Bytes: orderID, Valid: true},
	}); err != nil {
		return fmt.Errorf("log out movement: %w", err)
	}

	return nil
}

// AdjustStock adjusts inventory quantity (admin operation).
func (s *InventoryService) AdjustStock(ctx context.Context, variantID uuid.UUID, newQuantity int32, reason string) error {
	if newQuantity < 0 {
		return fmt.Errorf("quantity cannot be negative")
	}

	inv, err := s.store.GetInventoryByVariant(ctx, variantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Create new inventory record
			_, err = s.store.CreateInventoryItem(ctx, store.CreateInventoryItemParams{
				VariantID:         variantID,
				Quantity:          newQuantity,
				ReservedQuantity:  0,
				LowStockThreshold: pgtype.Int4{Int32: 5, Valid: true},
			})
			if err != nil {
				return fmt.Errorf("create inventory item: %w", err)
			}
		} else {
			return fmt.Errorf("get inventory: %w", err)
		}
	} else {
		_, err = s.store.UpdateInventoryQuantity(ctx, store.UpdateInventoryQuantityParams{
			VariantID: variantID,
			Quantity:  newQuantity,
		})
		if err != nil {
			return fmt.Errorf("update inventory: %w", err)
		}
	}

	// Determine movement type
	movementType := "adjustment"
	if reason == "" {
		reason = "Manual stock adjustment"
	}

	var oldQty int32
	if err == nil {
		oldQty = inv.Quantity
	}

	if newQuantity > oldQty {
		movementType = "in"
	} else if newQuantity < oldQty {
		movementType = "out"
	}

	if movementType != "adjustment" {
		qtyDiff := newQuantity - oldQty
		if qtyDiff < 0 {
			qtyDiff = -qtyDiff
		}
		if _, err := s.store.CreateStockMovement(ctx, store.CreateStockMovementParams{
			VariantID:    variantID,
			MovementType: movementType,
			Quantity:     qtyDiff,
			Reason:       pgtype.Text{String: reason, Valid: true},
		}); err != nil {
			return fmt.Errorf("log adjustment movement: %w", err)
		}
	} else {
		if _, err := s.store.CreateStockMovement(ctx, store.CreateStockMovementParams{
			VariantID:    variantID,
			MovementType: "adjustment",
			Quantity:     0,
			Reason:       pgtype.Text{String: reason, Valid: true},
		}); err != nil {
			return fmt.Errorf("log adjustment movement: %w", err)
		}
	}

	return nil
}

// GetInventoryItem retrieves an inventory item with product details.
func (s *InventoryService) GetInventoryItem(ctx context.Context, variantID uuid.UUID) (store.Inventory, error) {
	return s.store.GetInventoryByVariant(ctx, variantID)
}

// ListInventory retrieves paginated inventory with product details.
func (s *InventoryService) ListInventory(ctx context.Context, limit, offset int32) ([]store.ListInventoryRow, int64, error) {
	items, err := s.store.ListInventory(ctx, store.ListInventoryParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list inventory: %w", err)
	}

	count, err := s.store.CountInventory(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count inventory: %w", err)
	}

	return items, count, nil
}

// ListLowStockItems retrieves items at or below their low stock threshold.
func (s *InventoryService) ListLowStockItems(ctx context.Context) ([]store.ListLowStockItemsRow, error) {
	return s.store.ListLowStockItems(ctx)
}

// ListStockMovements retrieves stock movement history for a variant.
func (s *InventoryService) ListStockMovements(ctx context.Context, variantID uuid.UUID, limit, offset int32) ([]store.StockMovement, error) {
	return s.store.ListStockMovementsByVariant(ctx, store.ListStockMovementsByVariantParams{
		VariantID: variantID,
		Limit:     limit,
		Offset:    offset,
	})
}
