package fstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddSavesAndRegistersFile(t *testing.T) {
	store, err := New(t.TempDir(), 0, "http://example.test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer store.Close()

	info, err := store.Add(`bad:/\*?"<>|name.txt`, "", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if info.Name != "badname.txt" {
		t.Fatalf("info.Name = %q, want %q", info.Name, "badname.txt")
	}
	if info.Size != 5 {
		t.Fatalf("info.Size = %d, want 5", info.Size)
	}
	if info.MimeType != "text/plain" {
		t.Fatalf("info.MimeType = %q, want text/plain", info.MimeType)
	}

	got, path, ok := store.Get(info.ID)
	if !ok {
		t.Fatal("Get() did not find saved file")
	}
	if got.Name != info.Name {
		t.Fatalf("Get().Name = %q, want %q", got.Name, info.Name)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("saved data = %q, want hello", string(data))
	}
}

func TestAddAvoidsNameCollision(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "same.txt"), []byte("old"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store, err := New(dir, 0, "http://example.test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer store.Close()

	info, err := store.Add("same.txt", "", strings.NewReader("new"))
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if info.Name != "same.txt" {
		t.Fatalf("info.Name = %q, want same.txt", info.Name)
	}

	_, path, ok := store.Get(info.ID)
	if !ok {
		t.Fatal("Get() did not find saved file")
	}
	if filepath.Base(path) == "same.txt" {
		t.Fatalf("collision path = %q, want generated prefix", path)
	}
}

func TestCloseCanBeCalledMoreThanOnce(t *testing.T) {
	store, err := New(t.TempDir(), 0, "http://example.test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	store.Close()
	store.Close()
}
