// Package categories classifies files into intuitive storage groups.
package categories

import (
	"path/filepath"
	"strings"
)

// Category identifies a storage group. Its value is also its stable index.
type Category int

const (
	Video Category = iota
	Images
	Audio
	Archives
	Documents
	SourceCode
	Databases
	DataModels
	Binaries
	Caches
	Logs
	Fonts
	VirtualMachines
	Containers
	System
	Other
)

// Count is the number of categories and the size of category-indexed arrays.
const Count = int(Other) + 1

var names = [Count]string{
	Video:           "Video",
	Images:          "Images",
	Audio:           "Audio",
	Archives:        "Archives",
	Documents:       "Documents",
	SourceCode:      "Source code",
	Databases:       "Databases",
	DataModels:      "Data & models",
	Binaries:        "Binaries",
	Caches:          "Caches & temp",
	Logs:            "Logs",
	Fonts:           "Fonts",
	VirtualMachines: "VMs & images",
	Containers:      "Containers",
	System:          "System",
	Other:           "Other",
}

// String returns the human-readable category name.
func (c Category) String() string {
	if c < 0 || int(c) >= Count {
		return names[Other]
	}
	return names[c]
}

// Valid reports whether c is a known category.
func (c Category) Valid() bool { return c >= 0 && int(c) < Count }

// All returns every category in stable order.
func All() []Category {
	out := make([]Category, Count)
	for i := range out {
		out[i] = Category(i)
	}
	return out
}

var extensions = map[string]Category{
	// Video
	"mp4": Video, "mkv": Video, "avi": Video, "mov": Video, "wmv": Video,
	"flv": Video, "webm": Video, "m4v": Video, "mpg": Video, "mpeg": Video,
	"m2ts": Video, "mts": Video, "3gp": Video, "vob": Video,
	"ogv": Video, "rmvb": Video, "divx": Video,

	// Images
	"jpg": Images, "jpeg": Images, "png": Images, "gif": Images, "bmp": Images,
	"tif": Images, "tiff": Images, "webp": Images, "heic": Images, "heif": Images,
	"raw": Images, "cr2": Images, "cr3": Images, "nef": Images, "arw": Images,
	"dng": Images, "svg": Images, "ico": Images, "psd": Images, "xcf": Images,
	"ai": Images, "eps": Images, "avif": Images, "jxl": Images,

	// Audio
	"mp3": Audio, "flac": Audio, "wav": Audio, "aac": Audio, "ogg": Audio,
	"opus": Audio, "m4a": Audio, "wma": Audio, "aiff": Audio, "aif": Audio,
	"alac": Audio, "mid": Audio, "midi": Audio, "ape": Audio, "wv": Audio,

	// Archives and packages
	"zip": Archives, "tar": Archives, "gz": Archives, "bz2": Archives,
	"xz": Archives, "zst": Archives, "7z": Archives, "rar": Archives,
	"lz": Archives, "lzma": Archives, "cab": Archives, "tgz": Archives,
	"tbz2": Archives, "txz": Archives, "deb": Archives, "rpm": Archives,
	"pkg": Archives, "apk": Archives, "snap": Archives, "jar": Archives,
	"war": Archives, "ear": Archives, "z": Archives, "lzo": Archives,
	"pack": Archives, "idx": Archives, "rev": Archives, "bitmap": Archives,
	"midx": Archives,

	// Documents
	"pdf": Documents, "doc": Documents, "docx": Documents, "xls": Documents,
	"xlsx": Documents, "ppt": Documents, "pptx": Documents, "odt": Documents,
	"ods": Documents, "odp": Documents, "txt": Documents, "md": Documents,
	"rst": Documents, "rtf": Documents, "epub": Documents, "mobi": Documents,
	"tex": Documents, "csv": Documents, "tsv": Documents, "json": Documents,
	"yaml": Documents, "yml": Documents, "toml": Documents, "xml": Documents,
	"ini": Documents, "conf": Documents, "cfg": Documents, "properties": Documents,

	// Source code and markup
	"go": SourceCode, "rs": SourceCode, "c": SourceCode, "cc": SourceCode,
	"cpp": SourceCode, "cxx": SourceCode, "h": SourceCode, "hpp": SourceCode,
	"py": SourceCode, "js": SourceCode, "mjs": SourceCode, "cjs": SourceCode,
	"ts": SourceCode, "tsx": SourceCode, "jsx": SourceCode, "java": SourceCode,
	"kt": SourceCode, "kts": SourceCode, "swift": SourceCode, "rb": SourceCode,
	"php": SourceCode, "cs": SourceCode, "scala": SourceCode, "sh": SourceCode,
	"bash": SourceCode, "zsh": SourceCode, "fish": SourceCode, "lua": SourceCode,
	"r": SourceCode, "pl": SourceCode, "pm": SourceCode, "ex": SourceCode,
	"exs": SourceCode, "erl": SourceCode, "hs": SourceCode, "clj": SourceCode,
	"dart": SourceCode, "html": SourceCode, "htm": SourceCode, "css": SourceCode,
	"scss": SourceCode, "sass": SourceCode, "less": SourceCode, "sql": SourceCode,
	"vue": SourceCode, "svelte": SourceCode, "nim": SourceCode, "zig": SourceCode,
	"asm": SourceCode, "s": SourceCode, "v": SourceCode, "sol": SourceCode,

	// Databases
	"db": Databases, "sqlite": Databases, "sqlite3": Databases, "mdb": Databases,
	"bson": Databases, "ldb": Databases, "sst": Databases, "ibd": Databases,
	"myd": Databases, "couch": Databases, "realm": Databases, "dbf": Databases,

	// Datasets, tensors, and model artefacts
	"parquet": DataModels, "avro": DataModels, "orc": DataModels,
	"arrow": DataModels, "feather": DataModels, "tfrecord": DataModels,
	"npy": DataModels, "npz": DataModels, "pkl": DataModels,
	"pickle": DataModels, "joblib": DataModels, "pt": DataModels,
	"pth": DataModels, "ckpt": DataModels, "onnx": DataModels,
	"h5": DataModels, "hdf5": DataModels, "safetensors": DataModels,

	// Binaries and compiled artefacts
	"exe": Binaries, "dll": Binaries, "so": Binaries, "dylib": Binaries,
	"a": Binaries, "o": Binaries, "obj": Binaries, "lib": Binaries,
	"bin": Binaries, "elf": Binaries, "appimage": Binaries, "msi": Binaries,
	"class": Binaries, "pyc": Binaries, "pyo": Binaries, "wasm": Binaries,
	"ko": Binaries, "out": Binaries, "deb2": Binaries,

	// Logs and fonts
	"log": Logs, "journal": Logs,
	"ttf": Fonts, "otf": Fonts, "woff": Fonts, "woff2": Fonts, "eot": Fonts,
	"fon": Fonts, "fnt": Fonts,

	// Virtual machines and disk images
	"vmdk": VirtualMachines, "vdi": VirtualMachines, "vhd": VirtualMachines,
	"vhdx": VirtualMachines, "qcow": VirtualMachines, "qcow2": VirtualMachines,
	"iso": VirtualMachines, "img": VirtualMachines, "dmg": VirtualMachines,
	"vbox": VirtualMachines, "ova": VirtualMachines, "ovf": VirtualMachines,
	"rawdisk": VirtualMachines, "sparseimage": VirtualMachines,
	"wim": VirtualMachines, "swm": VirtualMachines, "vmem": VirtualMachines,
}

var cacheSegments = map[string]struct{}{
	".cache": {}, "cache": {}, "caches": {},
	"node_modules": {}, ".npm": {}, ".yarn": {}, ".pnpm-store": {},
	"__pycache__": {}, ".pytest_cache": {}, ".mypy_cache": {}, ".ruff_cache": {},
	".gradle": {}, ".m2": {}, ".ivy2": {}, ".cargo": {}, ".rustup": {},
	".venv": {}, "venv": {}, ".tox": {}, ".nox": {},
	".nuget": {}, ".pub-cache": {}, ".stack-work": {}, ".cabal": {},
	"tmp": {}, "temp": {}, ".tmp": {}, ".temp": {},
	"trash": {}, ".trash": {}, "trash-1000": {},
	".thumbnails": {}, ".thumbcache": {}, "thumbnails": {},
	"deriveddata": {}, ".terraform": {}, ".next": {}, ".nuxt": {},
	".parcel-cache": {}, ".turbo": {}, ".svelte-kit": {},
}

var logSegments = map[string]struct{}{
	"log": {}, "logs": {}, "journal": {},
}

var containerPrefixes = []string{
	"/var/lib/docker", "/var/lib/containers", "/var/lib/snapd", "/snap/",
	"/var/lib/flatpak", "/.local/share/flatpak", "/var/lib/containerd",
	"/var/lib/buildkit", "/run/containers",
}

var vmPrefixes = []string{
	"/var/lib/libvirt", "/virtualbox vms", "/vmware", "/.local/share/gnome-boxes",
	"/var/lib/virtualbox",
}

var systemPrefixes = []string{
	"/usr/", "/lib/", "/lib32/", "/lib64/", "/libx32/", "/bin/", "/sbin/",
	"/etc/", "/boot/", "/var/lib/", "/opt/", "/System/", "/Library/",
	"/Applications/", "/Windows/", "/Program Files/", "/Program Files (x86)/",
	"/ProgramData/",
}

// Classify returns the category for an absolute or relative path. Classification
// is deliberately deterministic and path-based so results are reproducible.
func Classify(path string) Category {
	p := strings.ToLower(filepath.ToSlash(path))
	if p == "" {
		return Other
	}

	for _, seg := range strings.Split(p, "/") {
		if _, ok := cacheSegments[seg]; ok {
			return Caches
		}
	}
	for _, seg := range strings.Split(p, "/") {
		if _, ok := logSegments[seg]; ok {
			return Logs
		}
	}
	if strings.Contains(p, "/.git/objects/") || strings.HasPrefix(p, ".git/objects/") {
		return Archives
	}
	for _, prefix := range containerPrefixes {
		if pathHasPrefix(p, prefix) {
			return Containers
		}
	}
	for _, prefix := range vmPrefixes {
		if pathHasPrefix(p, prefix) {
			return VirtualMachines
		}
	}
	for _, prefix := range systemPrefixes {
		if pathHasPrefix(p, prefix) {
			return System
		}
	}

	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(p)), ".")
	if c, ok := extensions[ext]; ok {
		return c
	}
	return Other
}

func pathHasPrefix(path, prefix string) bool {
	if path == prefix {
		return true
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return strings.HasPrefix(path, prefix)
}
