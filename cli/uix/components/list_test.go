package components

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
)

func TestListItems(t *testing.T) {
	items := []list.Item{
		NewListItem("Item 1", "Description 1"),
		NewListItem("Item 2", "Description 2"),
	}

	l := NewList(items, 80, 20)
	if len(l.Items()) != 2 {
		t.Errorf("expected 2 items, got %d", len(l.Items()))
	}
}

func TestListSetItems(t *testing.T) {
	items := []list.Item{
		NewListItem("Item 1", "Description 1"),
	}

	l := NewList(items, 80, 20)
	if len(l.Items()) != 1 {
		t.Error("should have 1 item")
	}

	newItems := []list.Item{
		NewListItem("New 1", "Desc 1"),
		NewListItem("New 2", "Desc 2"),
		NewListItem("New 3", "Desc 3"),
	}
	l.SetItems(newItems)
	if len(l.Items()) != 3 {
		t.Errorf("expected 3 items after SetItems, got %d", len(l.Items()))
	}
}

func TestListSelectedItem(t *testing.T) {
	items := []list.Item{
		NewListItem("Item 1", "Description 1"),
		NewListItem("Item 2", "Description 2"),
	}

	l := NewList(items, 80, 20)
	item := l.SelectedItem()
	if item == nil {
		t.Fatal("should have a selected item")
	}
	if item.(ListItem).Title() != "Item 1" {
		t.Errorf("expected 'Item 1', got '%s'", item.(ListItem).Title())
	}
}

func TestListSetSize(t *testing.T) {
	items := []list.Item{NewListItem("Item 1", "Desc 1")}
	l := NewList(items, 80, 20)

	l.SetSize(100, 30)
	if l.width != 100 {
		t.Errorf("expected width 100, got %d", l.width)
	}
	if l.height != 30 {
		t.Errorf("expected height 30, got %d", l.height)
	}

	// Test safe defaults
	l.SetSize(0, 0)
	if l.width != 80 {
		t.Errorf("expected width 80 for zero input, got %d", l.width)
	}
}

func TestListItem(t *testing.T) {
	item := NewListItem("Title", "Description")
	if item.Title() != "Title" {
		t.Errorf("expected 'Title', got '%s'", item.Title())
	}
	if item.Description() != "Description" {
		t.Errorf("expected 'Description', got '%s'", item.Description())
	}
	if item.FilterValue() != "Title" {
		t.Errorf("expected 'Title', got '%s'", item.FilterValue())
	}
}
