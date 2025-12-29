package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

const baseURL = "http://127.0.0.1:8000/api/modules"

func main() {
	time.Sleep(2 * time.Second) // Wait for server to be fully ready

	fmt.Println("Starting Inventory Verification...")

	// 1. Create Product
	productID, err := createRecord("products", map[string]interface{}{
		"name":  "Verification Product",
		"sku":   "VER-001",
		"stock": 100,
		"price": 50,
	})
	if err != nil {
		fmt.Printf("Failed to create product: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created Product: %s (Stock: 100)\n", productID)

	// 2. Create Sales Order
	orderID, err := createRecord("sales_orders", map[string]interface{}{
		"order_number": "SO-VERIFY-001",
		"date":         time.Now().Format("2006-01-02"),
		"status":       "Draft",
	})
	if err != nil {
		fmt.Printf("Failed to create order: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created Order: %s (Status: Draft)\n", orderID)

	// 3. Create Order Item
	_, err = createRecord("order_items", map[string]interface{}{
		"order_id":   orderID,
		"product_id": productID,
		"quantity":   5,
		"unit_price": 50,
	})
	if err != nil {
		fmt.Printf("Failed to create order item: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Created Order Item (Quantity: 5)")

	// 4. Update Order to Shipped
	fmt.Println("Updating Order to Shipped...")
	err = updateRecord("sales_orders", orderID, map[string]interface{}{
		"status": "Shipped",
	})
	if err != nil {
		fmt.Printf("Failed to update order: %v\n", err)
		os.Exit(1)
	}

	// Wait for automation to run
	fmt.Println("Waiting for automation...")
	time.Sleep(2 * time.Second)

	// 5. Verify Stock
	product, err := getRecord("products", productID)
	if err != nil {
		fmt.Printf("Failed to get product: %v\n", err)
		os.Exit(1)
	}

	stock := product["stock"]
	fmt.Printf("Product Stock post-shipment: %v\n", stock)

	// Check if stock is 95
	var stockVal float64
	switch v := stock.(type) {
	case float64:
		stockVal = v
	case int:
		stockVal = float64(v)
	case int32:
		stockVal = float64(v)
	case int64:
		stockVal = float64(v)
	default:
		fmt.Printf("Unknown type for stock: %T\n", stock)
	}

	if stockVal == 95 {
		fmt.Println("SUCCESS: Stock deducted correctly!")
	} else {
		fmt.Printf("FAILURE: Stock incorrect. Expected 95, got %v\n", stock)
		os.Exit(1)
	}
}

func createRecord(module string, data map[string]interface{}) (string, error) {
	jsonData, _ := json.Marshal(data)
	resp, err := http.Post(fmt.Sprintf("%s/%s/records", baseURL, module), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)

	// API returns { "id": "..." } or similar?
	// Based on `RecordController.CreateRecord`, it returns: `return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id, ...})`
	return fmt.Sprintf("%v", res["id"]), nil
}

func updateRecord(module, id string, data map[string]interface{}) error {
	jsonData, _ := json.Marshal(data)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/%s/records/%s", baseURL, module, id), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func getRecord(module, id string) (map[string]interface{}, error) {
	resp, err := http.Get(fmt.Sprintf("%s/%s/records/%s", baseURL, module, id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	return res, nil
}
