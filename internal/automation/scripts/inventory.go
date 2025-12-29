package scripts

import (
	"context"
	"fmt"
	"go-crm/internal/repository"
	"strconv"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ScriptRegistry holds all available scripts
var Registry = map[string]func(ctx context.Context, recordRepo repository.RecordRepository, moduleName string, recordID string) error{
	"deduct_inventory": DeductInventory,
}

// DeductInventory reduces product stock when an order is shipped
func DeductInventory(ctx context.Context, recordRepo repository.RecordRepository, moduleName string, recordID string) error {
	// 1. Validate logic only runs for sales_orders.
	// (Though the automation rule itself should safeguard this, double check doesn't hurt)
	if moduleName != "sales_orders" {
		return nil
	}

	// 2. Fetch the Order
	_, err := recordRepo.Get(ctx, moduleName, recordID)
	if err != nil {
		return fmt.Errorf("failed to fetch order: %v", err)
	}

	// 3. Find related Order Items
	// We need to find items where order_id = recordID
	// Since recordID in lookup is usually stored as {id, name} or just ID depending on implementation.
	// We'll search by exact string match or object match.
	// Since we don't have a complex query engine exposed here easily, we rely on List.
	// Filters: { "order_id": recordID }
	// NOTE: In many NoSQL lookup implementations, it might be stored as an object.
	// We might need to adjust filter logic in Repository if it requires precise object matching.
	// For now assumming filter works on string representation or exact value.

	filters := map[string]interface{}{
		"order_id": recordID,
	}

	items, err := recordRepo.List(ctx, "order_items", filters, 1000, 0, "created_at", 1)
	if err != nil {
		return fmt.Errorf("failed to fetch order items: %v", err)
	}

	// 4. Iterate and Deduct Stock
	for _, item := range items {
		// Get Quantity
		qtyVal, ok := item["quantity"]
		if !ok {
			continue
		}

		// Quantity might be float64 or int or string coming from JSON
		qty := toFloat(qtyVal)
		if qty <= 0 {
			continue
		}

		// Get Product ID
		prodVal, ok := item["product_id"]
		if !ok {
			continue
		}

		// Product ID might be a Lookup Object or just ID string
		productID := extractID(prodVal)
		if productID == "" {
			continue
		}

		// 5. Update Product
		// We need to fetch product first to get current stock?
		// Or can we do atomic $inc? Repository Update usually does $set.
		// If Reposiotry doesn't support $inc, we must Fetch + Update. // Race condition risk but OK for MVP.

		product, err := recordRepo.Get(ctx, "products", productID)
		if err != nil {
			fmt.Printf("Error fetching product %s: %v\n", productID, err)
			continue
		}

		currentStock := toFloat(product["stock"])
		newStock := currentStock - qty

		// Update
		err = recordRepo.Update(ctx, "products", productID, map[string]interface{}{
			"stock": newStock,
		})
		if err != nil {
			fmt.Printf("Error updating stock for product %s: %v\n", productID, err)
		}
	}

	return nil
}

func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}

func extractID(v interface{}) string {
	// If it's a map (Lookup object), get "id" or "_id"
	if m, ok := v.(map[string]interface{}); ok {
		if id, ok := m["id"]; ok {
			return fmt.Sprintf("%v", id)
		}
		if id, ok := m["_id"]; ok {
			return fmt.Sprintf("%v", id)
		}
		// Sometimes it might be directly stored?
	}
	// If it's primitive.ObjectID
	if oid, ok := v.(primitive.ObjectID); ok {
		return oid.Hex()
	}
	// If string
	s := fmt.Sprintf("%v", v)
	return s
}
