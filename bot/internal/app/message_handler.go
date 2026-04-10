package app

import (
	"context"
	"strings"

	"bot/internal/app/commands"
	"bot/internal/telegram"
)

func (a *App) handleMessage(ctx context.Context, msg telegram.Message) error {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return nil
	}

	if itemCode, warehouse, ok := commands.ExtractSelectedWarehouse(text); ok {
		itemName := a.itemNameFor(msg.Chat.ID, itemCode)
		a.rememberSelection(msg.Chat.ID, itemCode, itemName, warehouse)

		if statusMessageID, pending := a.consumeBatchChangePending(msg.Chat.ID); pending {
			a.upsertBatchStatusMessage(
				ctx,
				msg.Chat.ID,
				statusMessageID,
				formatBatchStatusText(
					SelectedContext{ItemCode: itemCode, ItemName: itemName, Warehouse: warehouse},
					0,
					"",
					0,
					"",
					"",
					"",
					"Item almashtirildi. Batch Start ni bosing.",
				),
			)
			a.deleteTrackedBatchPromptMessage(ctx, msg.Chat.ID)
			a.deleteTrackedWarehousePromptMessage(ctx, msg.Chat.ID)
			a.deleteMessageBestEffort(ctx, msg.Chat.ID, msg.MessageID, "delete selected-warehouse warning")
			return nil
		}

		if err := commands.HandleWarehouseSelected(ctx, a.deps(), msg.Chat.ID, itemCode, itemName, warehouse); err != nil {
			return err
		}
		a.deleteTrackedWarehousePromptMessage(ctx, msg.Chat.ID)
		a.deleteMessageBestEffort(ctx, msg.Chat.ID, msg.MessageID, "delete selected-warehouse warning")
		return nil
	}

	if itemCode, itemName, ok := commands.ExtractSelectedItem(text); ok {
		a.rememberItemChoice(msg.Chat.ID, itemCode, itemName)

		messageID, err := commands.HandleItemSelected(ctx, a.deps(), msg.Chat.ID, itemCode, itemName)
		if err != nil {
			return err
		}
		a.trackWarehousePromptMessage(ctx, msg.Chat.ID, messageID)
		a.deleteTrackedBatchPromptMessage(ctx, msg.Chat.ID)
		a.deleteMessageBestEffort(ctx, msg.Chat.ID, msg.MessageID, "delete selected-item warning")
		return nil
	}

	cmd := normalizeCommand(text)
	if cmd == "" {
		return nil
	}

	a.maybeDeleteCommandMessage(ctx, msg, cmd)

	switch cmd {
	case "/start":
		messageID, err := commands.HandleStart(ctx, a.deps(), msg)
		if err != nil {
			return err
		}
		a.trackStartInfoMessage(ctx, msg.Chat.ID, messageID)
		return nil
	case "/batch":
		messageID, err := commands.HandleBatch(ctx, a.deps(), msg)
		if err != nil {
			return err
		}
		a.trackBatchPromptMessage(ctx, msg.Chat.ID, messageID)
		a.deleteTrackedStartInfoMessage(ctx, msg.Chat.ID)
		a.deleteTrackedWarehousePromptMessage(ctx, msg.Chat.ID)
		a.clearBatchChangePending(msg.Chat.ID)
		a.clearSelection(msg.Chat.ID)
		return nil
	case "/log":
		return a.handleLogCommand(ctx, msg.Chat.ID)
	case "/epc":
		return a.handleEPCCommand(ctx, msg.Chat.ID)
	case "/calibrate":
		return a.handleCalibrateCommand(ctx, msg.Chat.ID, text)
	default:
		return a.tg.SendMessage(ctx, msg.Chat.ID, "Qo'llanadigan buyruqlar: /start, /batch, /log, /epc, /calibrate")
	}
}

func (a *App) maybeDeleteCommandMessage(ctx context.Context, msg telegram.Message, cmd string) {
	if !shouldDeleteUserCommand(cmd) {
		return
	}
	if msg.MessageID <= 0 {
		return
	}

	a.deleteMessageBestEffort(ctx, msg.Chat.ID, msg.MessageID, "deleteMessage warning")
}

func shouldDeleteUserCommand(cmd string) bool {
	switch cmd {
	case "/start", "/batch", "/log", "/epc", "/calibrate":
		return true
	default:
		return false
	}
}

func normalizeCommand(text string) string {
	parts := strings.Fields(strings.TrimSpace(text))
	if len(parts) == 0 {
		return ""
	}

	cmd := strings.ToLower(parts[0])
	if !strings.HasPrefix(cmd, "/") {
		return ""
	}
	if i := strings.Index(cmd, "@"); i > 0 {
		cmd = cmd[:i]
	}
	return cmd
}
