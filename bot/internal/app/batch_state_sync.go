package app

func (a *App) setBatchState(active bool, chatID int64, sel SelectedContext) {
	if a == nil || a.batchState == nil {
		return
	}
	if err := a.batchState.Set(active, chatID, sel.ItemCode, sel.ItemName, sel.Warehouse); err != nil {
		a.logBatch.Printf("batch state write error: %v", err)
	}
}

func (a *App) syncBatchStateFromSessions(chatHint int64) {
	a.batchMu.Lock()
	active := len(a.batchByChat) > 0
	if chatHint == 0 {
		for chatID := range a.batchByChat {
			chatHint = chatID
			break
		}
	}
	a.batchMu.Unlock()

	sel := SelectedContext{}
	if active && chatHint != 0 {
		if got, ok := a.getSelection(chatHint); ok {
			sel = got
		}
	}
	a.setBatchState(active, chatHint, sel)
}

func (a *App) setPrintRequest(epc string, qty float64, unit string, sel SelectedContext) {
	if a == nil || a.batchState == nil {
		return
	}
	if err := a.batchState.SetPrintRequest(epc, qty, unit, sel.ItemCode, sel.ItemName); err != nil {
		a.logBatch.Printf("print request write error: %v", err)
	}
}

func (a *App) clearPrintRequest() {
	if a == nil || a.batchState == nil {
		return
	}
	if err := a.batchState.ClearPrintRequest(); err != nil {
		a.logBatch.Printf("print request clear error: %v", err)
	}
}
