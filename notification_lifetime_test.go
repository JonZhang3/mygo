package mygo

import "testing"

// Retiring a delivered object's callbacks must keep notification-center entries
// clickable through the application's durable-ID handler, without rebinding a
// replacement notification that happens to share that ID.
func TestNotificationReleasePreservesReplacementAndDurableClicks(t *testing.T) {
	old := NewNotification(NotificationOptions{ID: "notification-target"})
	replacement := NewNotification(NotificationOptions{ID: old.ID()})
	notifications.Lock()
	saved := notifications.byID
	notifications.byID = map[string]*Notification{old.ID(): replacement}
	notifications.Unlock()
	defer func() { notifications.Lock(); notifications.byID = saved; notifications.Unlock() }()
	clicked := false
	replacement.OnClick(func() { clicked = true })
	old.Release()
	notificationClicked(replacement.ID())
	if !clicked {
		t.Fatal("retired object detached its replacement's click target")
	}
	replacement.Release()
	replacement.Release()
	clicked = false
	received := ""
	off := App.OnNotificationClick(func(id string) { received = id })
	defer off()
	notificationClicked(replacement.ID())
	if clicked || received != replacement.ID() {
		t.Fatal("release removed durable-ID navigation or kept retired callbacks")
	}
}
