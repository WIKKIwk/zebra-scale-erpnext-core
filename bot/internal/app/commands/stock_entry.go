package commands

import (
	"context"
	"fmt"
	"strings"

	"bot/internal/telegram"
)

const StockEntryCallbackMaterialReceipt = "stock:material_receipt"
const StockEntryCallbackBatchChangeItem = "stock:batch_change_item"
const StockEntryCallbackBatchStart = "stock:batch_start"
const StockEntryCallbackBatchStop = "stock:batch_stop"

func ExtractSelectedWarehouse(text string) (string, string, bool) {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) == 0 {
		return "", "", false
	}

	var itemCode string
	var warehouse string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)

		if strings.HasPrefix(lower, "item:") {
			itemCode = strings.TrimSpace(line[len("item:"):])
			continue
		}
		if strings.HasPrefix(lower, "ombor:") {
			warehouse = strings.TrimSpace(line[len("ombor:"):])
		}
	}

	if itemCode == "" || warehouse == "" {
		return "", "", false
	}
	return itemCode, warehouse, true
}

func HandleWarehouseSelected(ctx context.Context, deps Deps, chatID int64, itemCode, itemName, warehouse string) error {
	itemCode = strings.TrimSpace(itemCode)
	itemName = strings.TrimSpace(itemName)
	warehouse = strings.TrimSpace(warehouse)
	if itemCode == "" || warehouse == "" {
		return nil
	}
	if itemName == "" {
		itemName = itemCode
	}

	keyboard := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: "Material Receipt", CallbackData: StockEntryCallbackMaterialReceipt},
			},
		},
	}

	text := fmt.Sprintf("Item tanlandi: %s\nKod: %s\nOmbor tanlandi: %s\nStock entry tanlang:", itemName, itemCode, warehouse)
	return deps.TG.SendMessageWithInlineKeyboard(ctx, chatID, text, keyboard)
}

func BuildBatchControlKeyboard() *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: "Item almashtirish", CallbackData: StockEntryCallbackBatchChangeItem},
				{Text: "Batch Start", CallbackData: StockEntryCallbackBatchStart},
				{Text: "Batch Stop", CallbackData: StockEntryCallbackBatchStop},
			},
		},
	}
}
