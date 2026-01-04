package inventory

import (
	"context"
	"fmt"
	"time"

	"go-crm/internal/common/models"
	"go-crm/internal/features/module"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Local interface to avoid import cycle with record package
type RecordRepository interface {
	Create(ctx context.Context, moduleName string, product models.Product, data map[string]any) (any, error)
	Get(ctx context.Context, moduleName, id string) (map[string]any, error)
	List(ctx context.Context, moduleName string, filter map[string]any, accessFilter map[string]any, limit, offset int64, sortBy string, sortOrder int) ([]map[string]any, error)
	Update(ctx context.Context, moduleName, id string, data map[string]any) error
}

type InventoryService interface {
	HandleStockUpdate(ctx context.Context, moduleName string, recordData map[string]interface{}) error
}

type InventoryServiceImpl struct {
	RecordRepo RecordRepository
	ModuleRepo module.ModuleRepository
}

func NewInventoryService(recordRepo RecordRepository, moduleRepo module.ModuleRepository) InventoryService {
	return &InventoryServiceImpl{
		RecordRepo: recordRepo,
		ModuleRepo: moduleRepo,
	}
}

func (s *InventoryServiceImpl) HandleStockUpdate(ctx context.Context, moduleName string, recordData map[string]interface{}) error {
	// Identify the type of movement based on the module
	var moveType string
	var sign float64 // +1 or -1

	switch moduleName {
	case "purchase_receipts":
		moveType = "In"
		sign = 1.0
	case "shipments":
		moveType = "Out"
		sign = -1.0
	case "stock_adjustments":
		moveType = "Adjustment"
		// Sign depends on the quantity provided in the record (can be negative)
		sign = 1.0
	default:
		return nil // Not an inventory module
	}

	// Extract Items
	items, ok := recordData["items"].([]interface{})
	if !ok || len(items) == 0 {
		// Try subform format (sometimes it might be mapped differently?)
		// Assuming standard subform array of maps
		return nil
	}

	recordID, _ := recordData["_id"].(primitive.ObjectID)

	// Process each item
	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Get Product ID
		var productID string
		if pID, ok := itemMap["product_id"].(map[string]interface{}); ok {
			if id, ok := pID["id"].(string); ok {
				productID = id
			}
		} else if pID, ok := itemMap["product_id"].(primitive.ObjectID); ok {
			productID = pID.Hex()
		} else if pID, ok := itemMap["product_id"].(string); ok {
			productID = pID
		}

		if productID == "" {
			continue
		}

		// Get Quantity
		qty, _ := getFloat(itemMap["quantity"])
		if qty == 0 {
			continue
		}

		changeQty := qty * sign

		// Fix Product Lookup structure for Movement
		oid, _ := primitive.ObjectIDFromHex(productID)

		// Create Stock Movement Record
		movement := map[string]interface{}{
			"_id":              primitive.NewObjectID(),
			"product_id":       oid,
			"quantity":         changeQty,
			"type":             moveType,
			"reference_module": moduleName,
			"reference_id":     recordID.Hex(),
			"date":             time.Now(),
			"created_at":       time.Now(),
			"updated_at":       time.Now(),
		}

		// Product struct assumption: Empty product struct as we are creating raw record
		// The RecordRepository.Create signature requires a models.Product
		// We can pass an empty one or look it up.
		// Since we are inside the service, we might just pass empty.
		_, err := s.RecordRepo.Create(ctx, "stock_movements", models.ProductCRM, movement)
		// Need to know the Product Name of the module?
		// "stock_movements" is a system module. Is it under "crm" product?
		// modules.json doesn't specify product. It defaults to "crm"?
		// Let's assume "crm".
		if err != nil {
			fmt.Printf("Failed to create stock movement: %v\n", err)
			continue
		}

		// Update Inventory Module
		// 1. Try to find existing inventory record for this product
		filter := map[string]any{
			"product_id": oid,
		}
		// List(ctx, module, filter, accessFilter, limit, offset, sort, order)
		results, err := s.RecordRepo.List(ctx, "inventory", filter, nil, 1, 0, "", 0)

		if err == nil && len(results) > 0 {
			// Update Existing
			invRecord := results[0]
			invID := invRecord["_id"].(primitive.ObjectID).Hex()
			currentStock, _ := getFloat(invRecord["stock_quantity"])
			newStock := currentStock + changeQty

			updateData := map[string]interface{}{
				"stock_quantity":   newStock,
				"last_movement_at": time.Now(),
				"updated_at":       time.Now(),
			}

			err = s.RecordRepo.Update(ctx, "inventory", invID, updateData)
			if err != nil {
				fmt.Printf("Failed to update inventory: %v\n", err)
			}
		} else {
			// Create New Inventory Record
			newInv := map[string]interface{}{
				"_id":              primitive.NewObjectID(),
				"product_id":       oid,
				"stock_quantity":   changeQty,
				"average_cost":     0, // TODO: Calc avg cost
				"last_movement_at": time.Now(),
				"created_at":       time.Now(),
				"updated_at":       time.Now(),
			}
			_, err = s.RecordRepo.Create(ctx, "inventory", models.ProductCRM, newInv)
			if err != nil {
				fmt.Printf("Failed to create inventory record: %v\n", err)
			}
		}
	}

	return nil
}

func getFloat(unk interface{}) (float64, bool) {
	switch i := unk.(type) {
	case float64:
		return i, true
	case float32:
		return float64(i), true
	case int64:
		return float64(i), true
	case int:
		return float64(i), true
	case int32:
		return float64(i), true
	default:
		return 0, false
	}
}
