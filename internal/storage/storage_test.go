package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSplitScheme(t *testing.T) {
	tests := []struct {
		input      string
		wantScheme string
		wantRest   string
	}{
		{"gdrive://Video Clips", "gdrive://", "Video Clips"},
		{"s3://my-bucket/prefix", "s3://", "my-bucket/prefix"},
		{"dropbox:///Photos", "dropbox://", "/Photos"},
		{"onedrive://Documents/Media", "onedrive://", "Documents/Media"},
		{"/local/path", "", "/local/path"},
		{"C:\\Windows\\Path", "", "C:\\Windows\\Path"},
		{"", "", ""},
		{"noscheme", "", "noscheme"},
	}

	for _, tt := range tests {
		scheme, rest := SplitScheme(tt.input)
		if scheme != tt.wantScheme || rest != tt.wantRest {
			t.Errorf("SplitScheme(%q) = (%q, %q), want (%q, %q)",
				tt.input, scheme, rest, tt.wantScheme, tt.wantRest)
		}
	}
}

func TestCloudJoin(t *testing.T) {
	tests := []struct {
		parts []string
		want  string
	}{
		{[]string{"gdrive://Video Clips", "file.MOV"}, "gdrive://Video Clips/file.MOV"},
		{[]string{"s3://bucket/prefix", "key.mp4"}, "s3://bucket/prefix/key.mp4"},
		{[]string{"dropbox:///Photos", "vacation.jpg"}, "dropbox:///Photos/vacation.jpg"},
		{[]string{"/local/path", "file.mp4"}, "/local/path/file.mp4"},
		{[]string{}, ""},
		{[]string{"gdrive://root"}, "gdrive://root"},
	}

	for _, tt := range tests {
		got := CloudJoin(tt.parts...)
		if got != tt.want {
			t.Errorf("CloudJoin(%v) = %q, want %q", tt.parts, got, tt.want)
		}
	}
}

func TestCloudDir(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"gdrive://Video Clips/file.MOV", "gdrive://Video Clips"},
		{"s3://bucket/prefix/key.mp4", "s3://bucket/prefix"},
		{"dropbox:///Photos/vacation.jpg", "dropbox:///Photos"},
		{"/local/path/file.mp4", "/local/path"},
	}

	for _, tt := range tests {
		got := CloudDir(tt.input)
		if got != tt.want {
			t.Errorf("CloudDir(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCloudBase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"gdrive://Video Clips/file.MOV", "file.MOV"},
		{"s3://bucket/prefix/key.mp4", "key.mp4"},
		{"dropbox:///Photos/vacation.jpg", "vacation.jpg"},
		{"/local/path/file.mp4", "file.mp4"},
	}

	for _, tt := range tests {
		got := CloudBase(tt.input)
		if got != tt.want {
			t.Errorf("CloudBase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMediaExts(t *testing.T) {
	// Video formats
	for _, ext := range []string{".mp4", ".mov", ".avi", ".mkv", ".webm"} {
		if !MediaExts[ext] {
			t.Errorf("MediaExts missing video extension %q", ext)
		}
	}

	// Photo formats
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff", ".tif", ".heic", ".heif"} {
		if !MediaExts[ext] {
			t.Errorf("MediaExts missing photo extension %q", ext)
		}
	}

	// Non-media
	for _, ext := range []string{".txt", ".go", ".html", ".pdf", ".doc"} {
		if MediaExts[ext] {
			t.Errorf("MediaExts should not contain %q", ext)
		}
	}
}

func TestLocalStorageListFiles(t *testing.T) {
	dir := t.TempDir()

	// Create test files
	mediaFiles := []string{"clip.mp4", "photo.jpg", "video.mov"}
	nonMediaFiles := []string{"readme.txt", "config.json"}

	for _, name := range mediaFiles {
		os.WriteFile(filepath.Join(dir, name), []byte("test"), 0644)
	}
	for _, name := range nonMediaFiles {
		os.WriteFile(filepath.Join(dir, name), []byte("test"), 0644)
	}
	// Create a subdirectory (should be skipped)
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)

	ls := &LocalStorage{}
	files, err := ls.ListFiles(dir)
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}

	if len(files) != len(mediaFiles) {
		t.Errorf("ListFiles returned %d files, want %d", len(files), len(mediaFiles))
	}

	// Verify sorted alphabetically
	for i := 1; i < len(files); i++ {
		if files[i-1].Name > files[i].Name {
			t.Errorf("files not sorted: %q > %q", files[i-1].Name, files[i].Name)
		}
	}

	// Verify all returned files are media files
	for _, f := range files {
		ext := filepath.Ext(f.Name)
		if !MediaExts[ext] {
			t.Errorf("non-media file returned: %q", f.Name)
		}
	}
}

func TestLocalStorageListFilesEmpty(t *testing.T) {
	dir := t.TempDir()
	ls := &LocalStorage{}
	files, err := ls.ListFiles(dir)
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("ListFiles returned %d files for empty dir, want 0", len(files))
	}
}

func TestLocalStorageListFilesInvalidDir(t *testing.T) {
	ls := &LocalStorage{}
	_, err := ls.ListFiles("/nonexistent/directory/that/does/not/exist")
	if err == nil {
		t.Error("ListFiles should return error for nonexistent directory")
	}
}

func TestLocalStorageReadWriteFile(t *testing.T) {
	dir := t.TempDir()
	ls := &LocalStorage{}

	path := filepath.Join(dir, "test.json")
	content := []byte(`{"key": "value"}`)

	if err := ls.WriteFile(path, content); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := ls.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("ReadFile = %q, want %q", got, content)
	}
}

func TestLocalStorageRename(t *testing.T) {
	dir := t.TempDir()
	ls := &LocalStorage{}

	oldPath := filepath.Join(dir, "old.mp4")
	os.WriteFile(oldPath, []byte("video"), 0644)

	if err := ls.Rename(dir, "old.mp4", "new.mp4"); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	if ls.FileExists(oldPath) {
		t.Error("old file should not exist after rename")
	}
	if !ls.FileExists(filepath.Join(dir, "new.mp4")) {
		t.Error("new file should exist after rename")
	}
}

func TestLocalStorageCopyFile(t *testing.T) {
	dir := t.TempDir()
	ls := &LocalStorage{}

	srcPath := filepath.Join(dir, "source.mp4")
	dstPath := filepath.Join(dir, "copy.mp4")
	content := []byte("video content")
	os.WriteFile(srcPath, content, 0644)

	if err := ls.CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}

	if !ls.FileExists(srcPath) {
		t.Error("source should still exist after copy")
	}
	if !ls.FileExists(dstPath) {
		t.Error("destination should exist after copy")
	}

	got, _ := ls.ReadFile(dstPath)
	if string(got) != string(content) {
		t.Errorf("copied content = %q, want %q", got, content)
	}
}

func TestLocalStorageMoveFile(t *testing.T) {
	dir := t.TempDir()
	ls := &LocalStorage{}

	srcPath := filepath.Join(dir, "source.mp4")
	dstPath := filepath.Join(dir, "moved.mp4")
	os.WriteFile(srcPath, []byte("video"), 0644)

	if err := ls.MoveFile(srcPath, dstPath); err != nil {
		t.Fatalf("MoveFile: %v", err)
	}

	if ls.FileExists(srcPath) {
		t.Error("source should not exist after move")
	}
	if !ls.FileExists(dstPath) {
		t.Error("destination should exist after move")
	}
}

func TestLocalStorageMkdirAll(t *testing.T) {
	dir := t.TempDir()
	ls := &LocalStorage{}

	nested := filepath.Join(dir, "a", "b", "c")
	if err := ls.MkdirAll(nested); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	info, err := os.Stat(nested)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("created path is not a directory")
	}
}

func TestLocalStorageFileExists(t *testing.T) {
	dir := t.TempDir()
	ls := &LocalStorage{}

	path := filepath.Join(dir, "exists.mp4")
	os.WriteFile(path, []byte("test"), 0644)

	if !ls.FileExists(path) {
		t.Error("FileExists should return true for existing file")
	}
	if ls.FileExists(filepath.Join(dir, "nope.mp4")) {
		t.Error("FileExists should return false for nonexistent file")
	}
}

func TestLocalStorageIsLocal(t *testing.T) {
	ls := &LocalStorage{}
	if !ls.IsLocal() {
		t.Error("LocalStorage.IsLocal() should return true")
	}
}

func TestNormalizeBrowsePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"/", ""},
		{"Photos", "/Photos"},
		{"/Photos", "/Photos"},
		{"Photos/Vacation", "/Photos/Vacation"},
		{"/Photos/Vacation", "/Photos/Vacation"},
	}

	for _, tt := range tests {
		got := NormalizeBrowsePath(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeBrowsePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCopyFileLocal(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "dst.bin")

	content := []byte("binary data here")
	os.WriteFile(src, content, 0644)

	if err := CopyFileLocal(src, dst); err != nil {
		t.Fatalf("CopyFileLocal: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read copy: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("copy content mismatch")
	}
}
