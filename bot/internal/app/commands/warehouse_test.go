package commands

import (
	"context"
	"testing"

	"bot/internal/telegram"
)

func TestExtractSelectedItemCode(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{name: "normal", in: "Item: ITM-001\nNomi: Apple", want: "ITM-001", ok: true},
		{name: "spaces", in: " item:  GRENKI YASHIL  \nNomi: X", want: "GRENKI YASHIL", ok: true},
		{name: "invalid", in: "Nomi: Apple", want: "", ok: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ExtractSelectedItemCode(tc.in)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("got=(%q,%v) want=(%q,%v)", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestWarehouseInlineRoundtrip(t *testing.T) {
	seed := buildWarehouseInlineSeed("GRENKI YASHIL")
	req, ok := parseWarehouseInlineQuery(seed)
	if !ok {
		t.Fatalf("parse failed for seed=%q", seed)
	}
	if req.ItemCode != "GRENKI YASHIL" {
		t.Fatalf("item code mismatch: %q", req.ItemCode)
	}
	if req.Query != "" {
		t.Fatalf("query mismatch: %q", req.Query)
	}
}

func TestParseWarehouseInlineQuery_WithSearch(t *testing.T) {
	seed := buildWarehouseInlineSeed("ITM-001")
	seed = seed[:len(seed)-1] + "store"
	req, ok := parseWarehouseInlineQuery(seed)
	if !ok {
		t.Fatalf("parse failed for %q", seed)
	}
	if req.ItemCode != "ITM-001" {
		t.Fatalf("item code mismatch: %q", req.ItemCode)
	}
	if req.Query != "store" {
		t.Fatalf("query mismatch: %q", req.Query)
	}
}

type fakeTelegramService struct {
	text     string
	keyboard *telegram.InlineKeyboardMarkup
}

func (f *fakeTelegramService) SendMessage(ctx context.Context, chatID int64, text string) error {
	return nil
}

func (f *fakeTelegramService) SendMessageWithInlineKeyboard(ctx context.Context, chatID int64, text string, keyboard *telegram.InlineKeyboardMarkup) error {
	f.text = text
	f.keyboard = keyboard
	return nil
}

func (f *fakeTelegramService) SendMessageWithInlineKeyboardAndReturnID(ctx context.Context, chatID int64, text string, keyboard *telegram.InlineKeyboardMarkup) (int64, error) {
	f.text = text
	f.keyboard = keyboard
	return 1, nil
}

func (f *fakeTelegramService) AnswerInlineQuery(ctx context.Context, inlineQueryID string, results []telegram.InlineQueryResultArticle, cacheSeconds int) error {
	return nil
}

func (f *fakeTelegramService) AnswerCallbackQuery(ctx context.Context, callbackQueryID, text string) error {
	return nil
}

func TestHandleWarehouseSelected_ShowsBatchStart(t *testing.T) {
	tg := &fakeTelegramService{}
	deps := Deps{TG: tg}

	if err := HandleWarehouseSelected(context.Background(), deps, 1, "ITM-001", "Apple", "Stores - A"); err != nil {
		t.Fatalf("HandleWarehouseSelected error: %v", err)
	}

	if tg.keyboard == nil || len(tg.keyboard.InlineKeyboard) != 1 || len(tg.keyboard.InlineKeyboard[0]) != 1 {
		t.Fatalf("keyboard missing: %+v", tg.keyboard)
	}
	btn := tg.keyboard.InlineKeyboard[0][0]
	if btn.Text != "Batch Start" {
		t.Fatalf("button text = %q", btn.Text)
	}
	if btn.CallbackData != StockEntryCallbackBatchStart {
		t.Fatalf("callback = %q", btn.CallbackData)
	}
}
