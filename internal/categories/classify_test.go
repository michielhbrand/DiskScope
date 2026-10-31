package categories

import "testing"

func TestClassify(t *testing.T) {
	tests := []struct {
		path string
		want Category
	}{
		{"/home/u/Movies/holiday.MP4", Video},
		{"/home/u/Pictures/photo.JPG", Images},
		{"/home/u/Music/album/song.flac", Audio},
		{"/home/u/Downloads/archive.tar.gz", Archives},
		{"/home/u/Documents/notes.md", Documents},
		{"/home/u/src/main.go", SourceCode},
		{"/home/u/data/app.sqlite3", Databases},
		{"/home/u/models/random_forest.pkl", DataModels},
		{"/home/u/models/tensors.npz", DataModels},
		{"/home/u/repo/.git/objects/pack/pack-abc.pack", Archives},
		{"/home/u/repo/.git/objects/ab/cdef0123456789", Archives},
		{"/home/u/bin/tool.exe", Binaries},
		{"/var/log/syslog", Logs},
		{"/home/u/.cache/big-video.mp4", Caches},
		{"/home/u/project/node_modules/pkg/index.js", Caches},
		{"/var/lib/libvirt/images/disk.qcow2", VirtualMachines},
		{"/var/lib/docker/overlay2/abc/data.bin", Containers},
		{"/usr/lib/libc.so.6", System},
		{"/etc/hosts", System},
		{"/home/u/.local/share/fonts/MyFont.otf", Fonts},
		{"/home/u/file.unknownext", Other},
		{"", Other},
	}
	for _, tc := range tests {
		if got := Classify(tc.path); got != tc.want {
			t.Errorf("Classify(%q) = %s, want %s", tc.path, got, tc.want)
		}
	}
}

func TestCatalog(t *testing.T) {
	all := All()
	if len(all) != Count {
		t.Fatalf("All() returned %d categories, want %d", len(all), Count)
	}
	seen := map[string]bool{}
	for _, c := range all {
		if !c.Valid() {
			t.Errorf("category %d is not valid", c)
		}
		if c.String() == "" {
			t.Errorf("category %d has empty name", c)
		}
		if seen[c.String()] {
			t.Errorf("duplicate category name %q", c.String())
		}
		seen[c.String()] = true
	}
	if Category(999).String() != "Other" {
		t.Errorf("invalid category name = %q", Category(999).String())
	}
}
