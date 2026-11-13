package save

import (
	"context"
	"errors"
	"io/fs"
	"testing"
)

func TestUnit_SaveThenLoadReturnsTheData(t *testing.T) {
	// Arrange
	d := Dir{Path: t.TempDir()}
	ctx := context.Background()

	// Act
	err := d.Save(ctx, "slot_1", []byte(`{"money":250}`))
	data, loadErr := d.Load(ctx, "slot_1")

	// Assert
	if err != nil || loadErr != nil {
		t.Fatalf("save %v, load %v", err, loadErr)
	}
	if string(data) != `{"money":250}` {
		t.Fatalf("got %s", data)
	}
}

func TestUnit_EmptySlotIsNotExist(t *testing.T) {
	// Arrange
	d := Dir{Path: t.TempDir()}

	// Act
	_, err := d.Load(context.Background(), "nothing")

	// Assert
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("want not exist, got %v", err)
	}
}

func TestUnit_SlotNamesCannotLeaveTheFolder(t *testing.T) {
	// Arrange
	d := Dir{Path: t.TempDir()}

	// Act
	err := d.Save(context.Background(), "../escape", []byte("x"))

	// Assert
	if !errors.Is(err, ErrBadSlot) {
		t.Fatalf("want bad slot, got %v", err)
	}
}

func TestUnit_SaveReplacesTheOldData(t *testing.T) {
	// Arrange
	d := Dir{Path: t.TempDir()}
	ctx := context.Background()
	if err := d.Save(ctx, "a", []byte("old")); err != nil {
		t.Fatal(err)
	}

	// Act
	err := d.Save(ctx, "a", []byte("new"))
	data, _ := d.Load(ctx, "a")

	// Assert
	if err != nil || string(data) != "new" {
		t.Fatalf("err %v data %s", err, data)
	}
}

func TestUnit_GameNamesBecomeSafeFolders(t *testing.T) {
	// Arrange
	names := map[string]string{"Sunvale": "Sunvale", "My Game!": "My_Game_", "": "game"}

	for in, want := range names {
		// Act
		got := folderName(in)

		// Assert
		if got != want {
			t.Fatalf("%q: got %q, want %q", in, got, want)
		}
	}
}
