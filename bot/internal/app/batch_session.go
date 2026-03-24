package app

import "context"

func (a *App) startBatchSession(parent context.Context, chatID int64, run func(ctx context.Context)) {
	if chatID == 0 || run == nil {
		return
	}

	ctx, cancel := context.WithCancel(parent)

	a.batchMu.Lock()
	a.batchNextID++
	session := batchSession{id: a.batchNextID, cancel: cancel}
	prev, hasPrev := a.batchByChat[chatID]
	a.batchByChat[chatID] = session
	a.batchMu.Unlock()
	a.syncBatchStateFromSessions(chatID)

	if hasPrev && prev.cancel != nil {
		prev.cancel()
	}

	go func(chatID int64, sessionID int64) {
		run(ctx)
		a.batchMu.Lock()
		cur, ok := a.batchByChat[chatID]
		if ok && cur.id == sessionID {
			delete(a.batchByChat, chatID)
		}
		a.batchMu.Unlock()
		a.syncBatchStateFromSessions(chatID)
	}(chatID, session.id)
}

func (a *App) stopBatchSession(chatID int64) bool {
	if chatID == 0 {
		return false
	}

	a.batchMu.Lock()
	s, ok := a.batchByChat[chatID]
	if ok {
		delete(a.batchByChat, chatID)
	}
	a.batchMu.Unlock()
	a.syncBatchStateFromSessions(chatID)

	if ok && s.cancel != nil {
		s.cancel()
		return true
	}
	return false
}

func (a *App) hasBatchSession(chatID int64) bool {
	if chatID == 0 {
		return false
	}
	a.batchMu.Lock()
	_, ok := a.batchByChat[chatID]
	a.batchMu.Unlock()
	return ok
}

func (a *App) otherActiveBatchOwner(chatID int64) (int64, bool) {
	a.batchMu.Lock()
	defer a.batchMu.Unlock()
	for ownerChatID := range a.batchByChat {
		if ownerChatID != 0 && ownerChatID != chatID {
			return ownerChatID, true
		}
	}
	return 0, false
}

func (a *App) stopAllBatchSessions() {
	a.batchMu.Lock()
	cancels := make([]context.CancelFunc, 0, len(a.batchByChat))
	for _, s := range a.batchByChat {
		if s.cancel != nil {
			cancels = append(cancels, s.cancel)
		}
	}
	a.batchByChat = make(map[int64]batchSession)
	a.batchMu.Unlock()
	a.setBatchState(false, 0, SelectedContext{})

	for _, c := range cancels {
		c()
	}
}
