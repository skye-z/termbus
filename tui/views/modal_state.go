package views

import "github.com/termbus/termbus/tui/components/modal"

type modalState struct {
	active  bool
	confirm modal.ConfirmModal
}
