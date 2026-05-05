package whatsapp

import (
	"testing"
)

func TestClientBuildCatalogsURL(t *testing.T) {
	client := &Client{baseURL: "https://graph.facebook.com"}
	account := &Account{
		BusinessID: "123456789",
		APIVersion: "v18.0",
	}

	expected := "https://graph.facebook.com/v18.0/123456789/owned_product_catalogs"
	result := client.buildCatalogsURL(account)

	if result != expected {
		t.Errorf("buildCatalogsURL() = %v, want %v", result, expected)
	}
}

func TestClientBuildCatalogProductsURL(t *testing.T) {
	client := &Client{baseURL: "https://graph.facebook.com"}
	account := &Account{
		APIVersion: "v18.0",
	}
	catalogID := "catalog_123"

	expected := "https://graph.facebook.com/v18.0/catalog_123/products"
	result := client.buildCatalogProductsURL(account, catalogID)

	if result != expected {
		t.Errorf("buildCatalogProductsURL() = %v, want %v", result, expected)
	}
}

func TestClientBuildProductURL(t *testing.T) {
	client := &Client{baseURL: "https://graph.facebook.com"}
	account := &Account{
		APIVersion: "v18.0",
	}
	productID := "product_456"

	expected := "https://graph.facebook.com/v18.0/product_456"
	result := client.buildProductURL(account, productID)

	if result != expected {
		t.Errorf("buildProductURL() = %v, want %v", result, expected)
	}
}
