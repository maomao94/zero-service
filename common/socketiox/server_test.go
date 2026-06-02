package socketiox

import (
	"fmt"
	"testing"
)

func TestBuildRoomsPageResSortsAndPaginates(t *testing.T) {
	rooms := []string{"room:c", "room:a", "room:b"}

	got := buildRoomsPageRes(rooms, 1, 2)

	if got.Total != 3 {
		t.Fatalf("total = %d, want 3", got.Total)
	}
	if got.Page != 1 || got.PageSize != 2 || got.TotalPages != 2 {
		t.Fatalf("page fields = page %d pageSize %d totalPages %d, want 1/2/2", got.Page, got.PageSize, got.TotalPages)
	}
	wantRooms := []string{"room:a", "room:b"}
	if len(got.Rooms) != len(wantRooms) {
		t.Fatalf("rooms len = %d, want %d", len(got.Rooms), len(wantRooms))
	}
	for i, want := range wantRooms {
		if got.Rooms[i] != want {
			t.Fatalf("rooms[%d] = %q, want %q", i, got.Rooms[i], want)
		}
	}
}

func TestBuildRoomsPageResNormalizesDefaults(t *testing.T) {
	rooms := []string{"room:a"}

	got := buildRoomsPageRes(rooms, 0, 0)

	if got.Page != defaultRoomsPage {
		t.Fatalf("page = %d, want %d", got.Page, defaultRoomsPage)
	}
	if got.PageSize != defaultRoomsPageSize {
		t.Fatalf("pageSize = %d, want %d", got.PageSize, defaultRoomsPageSize)
	}
	if got.Total != 1 || got.TotalPages != 1 || len(got.Rooms) != 1 {
		t.Fatalf("result = total %d totalPages %d rooms %d, want 1/1/1", got.Total, got.TotalPages, len(got.Rooms))
	}
}

func TestBuildRoomsPageResCapsPageSize(t *testing.T) {
	rooms := make([]string, maxRoomsPageSize+1)
	for i := range rooms {
		rooms[i] = fmt.Sprintf("room:%03d", i)
	}

	got := buildRoomsPageRes(rooms, 1, maxRoomsPageSize+100)

	if got.PageSize != maxRoomsPageSize {
		t.Fatalf("pageSize = %d, want %d", got.PageSize, maxRoomsPageSize)
	}
	if len(got.Rooms) != maxRoomsPageSize {
		t.Fatalf("rooms len = %d, want %d", len(got.Rooms), maxRoomsPageSize)
	}
}

func TestBuildRoomsPageResOutOfRangePage(t *testing.T) {
	rooms := []string{"room:a"}

	got := buildRoomsPageRes(rooms, 2, 1)

	if got.Total != 1 || got.Page != 2 || got.PageSize != 1 || got.TotalPages != 1 {
		t.Fatalf("page result = total %d page %d pageSize %d totalPages %d, want 1/2/1/1", got.Total, got.Page, got.PageSize, got.TotalPages)
	}
	if len(got.Rooms) != 0 {
		t.Fatalf("rooms len = %d, want 0", len(got.Rooms))
	}
}

func TestVisibleSessionRoomsFiltersSocketIdRoom(t *testing.T) {
	got := visibleSessionRooms([]string{"socket-1", "room:a", "room:b"}, "socket-1")

	wantRooms := []string{"room:a", "room:b"}
	if len(got) != len(wantRooms) {
		t.Fatalf("rooms len = %d, want %d", len(got), len(wantRooms))
	}
	for i, want := range wantRooms {
		if got[i] != want {
			t.Fatalf("rooms[%d] = %q, want %q", i, got[i], want)
		}
	}
}
