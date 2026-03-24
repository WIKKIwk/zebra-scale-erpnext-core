package telegram

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInlineKeyboardButton_EmptySwitchInlineQueryCurrentChatIsOmitted(t *testing.T) {
	k := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{{
			{Text: "Item tanlash"},
		}},
	}

	b, err := json.Marshal(k)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	const bad = `"switch_inline_query_current_chat"`
	if strings.Contains(string(b), bad) {
		t.Fatalf("did not expect %s in payload, got: %s", bad, string(b))
	}
}

func TestInlineKeyboardButton_CallbackDataIsSerialized(t *testing.T) {
	k := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{{
			{Text: "Material Receipt", CallbackData: "stock:material_receipt"},
		}},
	}

	b, err := json.Marshal(k)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	const want = `"callback_data":"stock:material_receipt"`
	if !strings.Contains(string(b), want) {
		t.Fatalf("expected %s in payload, got: %s", want, string(b))
	}
}

func TestSendDocument_MultipartPayload(t *testing.T) {
	t.Parallel()

	var gotMethod string
	var gotPath string
	var gotChatID string
	var gotCaption string
	var gotFilename string
	var gotData []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path

		if err := r.ParseMultipartForm(2 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}

		gotChatID = r.FormValue("chat_id")
		gotCaption = r.FormValue("caption")

		f, h, err := r.FormFile("document")
		if err != nil {
			t.Fatalf("form file: %v", err)
		}
		defer f.Close()

		gotFilename = h.Filename
		gotData, err = io.ReadAll(f)
		if err != nil {
			t.Fatalf("read file: %v", err)
		}

		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := New("tok123")
	c.baseURL = srv.URL

	err := c.SendDocument(context.Background(), 12345, "logs.zip", []byte("abc123"), "caption text")
	if err != nil {
		t.Fatalf("SendDocument error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/bottok123/sendDocument" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotChatID != "12345" {
		t.Fatalf("chat_id = %q", gotChatID)
	}
	if gotCaption != "caption text" {
		t.Fatalf("caption = %q", gotCaption)
	}
	if gotFilename != "logs.zip" {
		t.Fatalf("filename = %q", gotFilename)
	}
	if string(gotData) != "abc123" {
		t.Fatalf("document data = %q", string(gotData))
	}
}
