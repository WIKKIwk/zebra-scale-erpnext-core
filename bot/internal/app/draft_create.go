package app

import (
	"fmt"
	"strings"

	"bot/internal/erp"
)

const maxDuplicateBarcodeRetries = 5

func createDraftWithFreshEPC(nextEPC func() string, create func(string) (erp.StockEntryDraft, error)) (string, erp.StockEntryDraft, error) {
	if nextEPC == nil || create == nil {
		return "", erp.StockEntryDraft{}, fmt.Errorf("draft create helper dependency bo'sh")
	}

	var lastErr error
	var lastEPC string
	for attempt := 0; attempt < maxDuplicateBarcodeRetries; attempt++ {
		epc := strings.ToUpper(strings.TrimSpace(nextEPC()))
		if epc == "" {
			return "", erp.StockEntryDraft{}, fmt.Errorf("epc generator bo'sh qiymat qaytardi")
		}
		lastEPC = epc

		draft, err := create(epc)
		if err == nil {
			return epc, draft, nil
		}
		lastErr = err
		if !erp.IsDuplicateBarcodeError(err) {
			return epc, erp.StockEntryDraft{}, err
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("duplicate retry exhausted")
	}
	return lastEPC, erp.StockEntryDraft{}, fmt.Errorf("duplicate barcode retry exhausted after %d attempts: %w", maxDuplicateBarcodeRetries, lastErr)
}
